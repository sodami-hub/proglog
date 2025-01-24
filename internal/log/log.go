package log

import (
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	api "github.com/sodami-hub/proglog/api/v1"
)

type Log struct {
	mu     sync.Mutex
	Dir    string
	Config Config

	activeSegment *segment
	segments      []*segment
}

// NewLog 함수는 로그 파일을(세그먼트, 인덱스) 새로 만든다는 의미가 아니라
// 서비스를 시작한다는 의미이다.
func NewLog(dir string, c Config) (*Log, error) {
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = 1024
	}
	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = 1024
	}
	l := &Log{
		Dir:    dir,
		Config: c,
	}

	return l, l.setup()
}

func (l *Log) setup() error {
	files, err := os.ReadDir(l.Dir)
	if err != nil {
		return err
	}
	var baseOffsets []uint64
	for _, file := range files {
		offStr := strings.TrimSuffix( // 파일 이름에서 확장자 제거
			file.Name(),           // 파일이름
			path.Ext(file.Name()), // 제거하고자 하는 Suffix -> 확장자
		)
		// 문자열을 10진수(base 10)로 해석하여 부호 없는 정수 uin64로 변환한다.
		off, _ := strconv.ParseUint(offStr, 10, 0)
		baseOffsets = append(baseOffsets, off)
	}
	sort.Slice(baseOffsets, func(i, j int) bool {
		return baseOffsets[i] < baseOffsets[j]
	})

	for i := 0; i < len(baseOffsets); i++ {
		if err = l.newSegment(baseOffsets[i]); err != nil {
			return err
		}
		// 베이스 오프셋은 index와 store 두 파일을 중복해서 담고 있기에
		// 같은 값이 하나 더 있다. 그래서 한 번 건너뛴다.
		i++
	}
	if l.segments == nil {
		if err = l.newSegment(
			l.Config.Segment.InitialOffset,
		); err != nil {
			return err
		}
	}
	return nil
}

func (l *Log) Append(record *api.Record) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.activeSegment.IsMaxed() {
		off := l.activeSegment.nextOffset
		if err := l.newSegment(off); err != nil {
			return 0, err
		}
	}
	return l.activeSegment.Append(record)
}

func (l *Log) Read(off uint64) (*api.Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var s *segment
	for _, segment := range l.segments {
		if segment.baseOffset <= off && off < segment.nextOffset {
			s = segment
			break
		}
	}
	if s == nil || s.nextOffset <= off {
		return nil, fmt.Errorf("offset out of range: %d", off)
	}
	return s.Read(off)
}

// 로그의 모든 세그먼트를 닫는다.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, segment := range l.segments {
		if err := segment.Close(); err != nil {
			return err
		}
	}
	return nil
}

// 로그를 닫고 데이터를 지운다.
func (l *Log) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}
	return os.RemoveAll(l.Dir)
}

// 로그를 제거하고 이를 대체할 새로운 로그를 생성한다.
func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return err
	}
	return l.setup()
}

// 아래 두개의 메서드는 로그에 저장된 오프셋의 범위를 알려준다. 복제 기능 지원이나 클러스터 조율을 할 때 이러한 정보가 필요하다.
func (l *Log) LowestOffset() (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.segments[0].baseOffset, nil
}

func (l *Log) HighestOffset() (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	off := l.segments[len(l.segments)-1].nextOffset
	if off == 0 {
		return 0, nil
	}
	return off - 1, nil
}

// Truncate 메서드는 가장 큰 오프셋이 가장 작은 오프셋(매개변수 값)보다 작은 세그먼트를 찾아 제거한다.
// 즉, 특정 시점보다 오래된 세그먼트를 지우는 메서드이다.
func (l *Log) Truncate(lowest uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var segments []*segment
	for _, s := range l.segments {
		if s.nextOffset <= lowest+1 {
			if err := s.Remove(); err != nil {
				return err
			}
			continue
		}
		segments = append(segments, s)
	}
	l.segments = segments
	return nil
}

/*
Reader 메서드는 io.Reader 인터페이스 자료형을 리턴하여 전체 로그를 읽도록 한다. 조율한 합의를 구현할 때와 스냅숏,
로그 복원 기능을 지원할 때 필요하다. Reader() 메서드는 io.MultiReader()를 호출하여 세그먼트의 스토어들을 하나로 모은다.
*/
func (l *Log) Reader() io.Reader {
	l.mu.Lock()
	defer l.mu.Unlock()
	readers := make([]io.Reader, len(l.segments))
	for i, segment := range l.segments {
		readers[i] = &originReader{segment.store, 0}
	}
	return io.MultiReader(readers...)
}

type originReader struct {
	*store
	off int64
}

func (o *originReader) Read(p []byte) (int, error) {
	n, err := o.ReadAt(p, o.off)
	o.off += int64(n)
	return n, err
}

func (l *Log) newSegment(off uint64) error {
	s, err := newSegment(l.Dir, off, l.Config)
	if err != nil {
		return err
	}
	l.segments = append(l.segments, s)
	l.activeSegment = s
	return nil
}
