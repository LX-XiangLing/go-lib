package guid

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var (
	timeEpoch        time.Time
	incrBits               = 32
	int32Max         int64 = 2147483647
	ErrorInvalidStep       = errors.New("invalid step")
)

type IdData struct {
	id   int64
	rest int64
}

type IdSource interface {
	NextId(ctx context.Context, kind string, step int64) (id int64, err error)
}

// IdGenerator 弱中心化全局唯一ID生成算法
type IdGenerator struct {
	idgen IdSource
	kind  string
	step  int64
	steps []int64

	// 是否填充相对时间
	relts bool

	data *IdData
	lock sync.Mutex
}

func init() {
	var err error
	timeEpoch, err = time.ParseInLocation("2006-01-02 15:04:05", "2021-08-01 00:00:00", time.Local)
	if err != nil {
		panic(err)
	}
	rand.Seed(time.Now().UnixNano())
}

func New(idgen IdSource, kind string) *IdGenerator {
	return &IdGenerator{
		idgen: idgen,
		kind:  kind,
		data:  &IdData{},
		steps: []int64{10000, 20000, 40000, 80000, 80000},
		relts: true,
	}
}

// SetStep set the step and return the old step
func (ig *IdGenerator) SetStep(steps ...int64) []int64 {
	old := ig.steps
	ig.steps = steps
	return old
}

func (ig *IdGenerator) FillWithRelTs(b bool) *IdGenerator {
	ig.relts = b
	return ig
}

func (ig *IdGenerator) Id(ctx context.Context, kinds ...string) (int64, error) {
	return ig.GenId(ctx, 1)
}

func (ig *IdGenerator) NextId(ctx context.Context) (int64, error) {
	return ig.GenId(ctx, 1)
}

func (ig *IdGenerator) GenId(ctx context.Context, step int64) (int64, error) {
	if step < 1 || step > 9999 {
		return 0, ErrorInvalidStep
	}
	kind := ig.kind
	var ret int64
	var err error

	rest := atomic.AddInt64(&ig.data.rest, -step)
	if rest < 0 {
		ig.lock.Lock()
		defer ig.lock.Unlock()
		if atomic.LoadInt64(&ig.data.rest) > 0 {
			return ig.NextId(ctx)
		}
		n := rand.Intn(len(ig.steps))
		ig.step = ig.steps[n]
		var id int64
		id, err = ig.idgen.NextId(ctx, kind, ig.step)
		if err == nil {
			if id > 0 {
				ig.data.id = id
				atomic.StoreInt64(&ig.data.rest, ig.step)
				return ig.NextId(ctx)
			} else {
				return 0, fmt.Errorf("idgen returned id %v", id)
			}
		} else {
			return 0, err
		}
	} else {
		ret = ig.data.id - rest
	}
	if ig.relts {
		ret = int32Max & ret
		// 第 1 bit 保留
		// 第 2~32 bit 为相对时间戳，确保 ID 随时间单调
		// 第 33~64 bit 为全局递增序列
		// 以保证在可预见的计算能力下：
		// 	- 分布式系统内相同的时间戳具有不同 ID
		// 	- 进程内时钟回拨 ID 不重复
		ts := int64(time.Since(timeEpoch).Seconds())
		ret = (ts<<incrBits | ret)
	}
	return ret, nil
}

func Decode(id int64) (relTs time.Time, seqId int64) {
	seqId = int32Max & id
	dur := id >> incrBits
	relTs = timeEpoch.Add(time.Duration(dur) * time.Second)
	return
}
