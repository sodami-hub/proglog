package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

// 레코드 크기와 인덱스 항목을 저장할 때의 인코딩을 정의
var enc = binary.BigEndian

// 레코드 길이를 저장하는 바이트 개수를 정의한 것 - uint64 -> 8byte
const lenWidth = 8

type store struct {
	*os.File // *os.File 을 임베딩했다. os.File의 모든 메서드와 필드를 사용할 수 있다.
	mu       sync.Mutex
	buf      *bufio.Writer
	size     uint64
}

func newStore(f *os.File) (*store, error) {
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	size := uint64(fi.Size())
	return &store{
		File: f, // 그런데 이 File 이라는 필드명은 어디서 온건지...
		size: size,
		buf:  bufio.NewWriter(f),
	}, nil
}

/*
Append 메서는 저장할 레코드를 받아서 store 구조체에 저장(레코드 크기+레코드)하고,
실제 저장한 데이터 크기(n byte), 저장하기전 store의 크기(pos byte)를 반환한다.
*/
func (s *store) Append(p []byte) (n uint64, pos uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pos = s.size
	// 레코드의 길이를 저장 uint64 타입이므로 8바이트를 차지한다.(lenWidth)
	if err := binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, err
	}
	w, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}
	w += lenWidth       // 실제 저장한 데이터의 크기(w byte)에 p의 길이를 저장한 uint64타입의 크기인 8을 더한다.
	s.size += uint64(w) // 실제로 레코드를 저장하기 위해서 사용한 크기(w)를 현재 사이즈에 더해서 크기를 갱신한다.
	return uint64(w), pos, nil
}

/*
Read 메서드는 pos를 받아서 레코드의 크기를 읽어내고
그 크기만큼 실제 레코드를 반환한다.
*/
func (s *store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return nil, err
	}

	// 레코드의 크기를 읽기위한 부분
	size := make([]byte, lenWidth)
	// pos 에서부터 size크기만큼 읽는다.
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}

	// 앞에서 가져온 레코드의 크기를 통해서 파일에서 실제 레코드만 읽어낸다.
	b := make([]byte, enc.Uint64(size))
	if _, err := s.File.ReadAt(b, int64(pos+lenWidth)); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return 0, err
	}
	return s.File.ReadAt(p, off)
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return err
	}
	return s.File.Close()
}
