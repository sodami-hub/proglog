package log

import (
	"fmt"
	"os"
	"path"

	api "github.com/sodami-hub/proglog/api/v1"
	"google.golang.org/protobuf/proto"
	// _ "google.golang.org/protobuf/proto"
)

type segment struct {
	store                  *store
	index                  *index
	baseOffset, nextOffset uint64
	config                 Config
}

func newSegment(dir string, baseOffset uint64, c Config) (*segment, error) {
	s := &segment{
		baseOffset: baseOffset,
		config:     c,
	}
	var err error
	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store")),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}
	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".offset")),
		os.O_RDWR|os.O_CREATE,
		0644,
	)
	if err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err
	}
	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset // index 파일의 사이즈가 0 -> baseOffset 부터 오프셋 시작
	} else {
		s.nextOffset = baseOffset + uint64(off) + 1 // index의 마지막 위치가 off로 넘어옴 다음 오프셋을 구함.
	}
	return s, nil
}

func (s *segment) Append(record *api.Record) (offset uint64, err error) {
	cur := s.nextOffset
	record.Offset = cur
	p, err := proto.Marshal(record)
	if err != nil {
		return 0, err
	}

	_, pos, err := s.store.Append(p)
	if err != nil {
		return 0, err
	}
	if err = s.index.Write(
		// 인덱스의 오프셋은 베이스 오프셋에서의 상댓값이다.
		uint32(s.nextOffset-uint64(s.baseOffset)),
		pos,
	); err != nil {
		return 0, err
	}
	s.nextOffset++
	return cur, nil
}

// 매개변수 off는 절대값으로 넘어옴 0~ .... // 반면에 각 인덱스 파일의 순서는 0부터 시작됨.
func (s *segment) Read(off uint64) (*api.Record, error) {
	_, pos, err := s.index.Read(int64(off - s.baseOffset)) // 인덱스의 오프셋은 베이스오프셋에서의 상댓값이기 때문에...
	if err != nil {
		return nil, err
	}
	p, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}
	record := &api.Record{}
	err = proto.Unmarshal(p, record)
	return record, err
}

/*
세그먼트 스토어 또는 인덱스가 최대 크기에 도달했는지를 리턴한다. 추가하는 레코드의 저장 바이트는 가변이기에 현재 크기가 저장 바이트 제한을
넘지 않으면 되고, 추가하는 레코드에 대한 인덱스 바이트는 고정적이기에(entWidth) 현재의 크기에 인덱스 하나를 추가했을 때 인덱스 제한을 넘지 않아야 한다.
이 메서드를 사용해서 세그먼트의 용량이 가득 찼는지 확인하여 로그가 새로운 세그먼트를 만들지 판단한다.
*/
func (s *segment) IsMaxed() bool {
	return s.store.size >= s.config.Segment.MaxStoreBytes || s.index.size+entWidth >= s.config.Segment.MaxIndexBytes
}

func (s *segment) Remove() error {
	if err := s.Close(); err != nil {
		return err
	}
	if err := os.Remove(s.index.Name()); err != nil {
		return err
	}
	if err := os.Remove(s.store.Name()); err != nil {
		return err
	}
	return nil
}

func (s *segment) Close() error {
	if err := s.index.Close(); err != nil {
		return err
	}
	if err := s.store.Close(); err != nil {
		return err
	}
	return nil
}
