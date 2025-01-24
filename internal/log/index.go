package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

// 인덱스 항목 내의 바이트 수를 정의
var (
	offWidth uint64 = 4                   // 레코드 오프셋
	posWidth uint64 = 8                   // 스토어 파일에서의 위치 -> store구조체의 File에 이 pos의 위치에서부터 8바이트는 해당 레코드의 길이가 저장돼있고, 이후에 실제 데이터가 있다.
	entWidth        = offWidth + posWidth // 오프셋이 가리키는 위치 / 인덱스 한 항목의 크기
)

type index struct {
	file *os.File    // 파일
	mmap gommap.MMap // 메모리 맵 파일
	size uint64
}

/*
newIndex 함수는 매개변수인 *os.File에 해당하는 파일을 위한 인덱스를 생성한다.
*/
func newIndex(f *os.File, c Config) (*index, error) {
	idx := &index{
		file: f,
	}
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	idx.size = uint64(fi.Size())
	if err = os.Truncate(
		f.Name(), int64(c.Segment.MaxIndexBytes),
	); err != nil {
		return nil, err
	}
	if idx.mmap, err = gommap.Map(
		idx.file.Fd(),
		gommap.PROT_READ|gommap.PROT_WRITE,
		gommap.MAP_SHARED,
	); err != nil {
		return nil, err
	}
	return idx, err
}

/*
메모리 맵 파일과 실제 팡리의 데이터가 확실히 동기화, 실제 파일 콘텐츠가 안정적인 저장소에 플러시된다.
그리고 실제 데이터가 있는 만큼만 잘라내고(truncate) 파일을 닫는다.

서비스를 시작하면, 서비스는 다음 레코드를 로그의 어디에 추가할지 오프셋을 알아야 한다. 마지막 항목의 인덱스를 찾아보면 다음 레코드의 오프셋을 알 수 있따.
인덱스 파일의 마지막 12바이트를 읽으면 된다. 하지만 메모리 맵 파일을 사용하기 위해 파일을 최대 큭기로 늘리면 이 방법을 사용할 수 없다.
메모리 맵 파일은 생성한 다음 크기를 바꿀 수 없기에 미리 필요한 크기로 만들어야 된다. 즉, 파일의 뒷쪽에 빈공간이 있다는 의미이다.
그렇기 때문에 서비스를 종료할 때 데이타의 크기에 맞춰서 파일을 자르고 서비스를 종료한다.
*/
func (i *index) Close() error {

	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}
	if err := i.file.Sync(); err != nil {
		return err
	}
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}
	return i.file.Close()
}

/*
Read() 메서드는 매개변수로 오프셋(저장 순서)을 받아서 해당하는 레코드의 저장 파일 내 위치를 리턴한다.
여기서 오프셋은 해당 세그먼트의 베이스 오프셋의 상댓값이다.
*/
func (i *index) Read(in int64) (out uint32, pos uint64, err error) {
	if i.size == 0 {
		return 0, 0, io.EOF
	}
	if in == -1 {
		out = uint32((i.size / entWidth) - 1) // 제일 마지막 인덱스를 가져온다.
	} else {
		out = uint32(in)
	}
	pos = uint64(out) * entWidth // 저장순서 * 크기 -> 실제 위치
	if i.size < pos+entWidth {
		return 0, 0, io.EOF
	}
	out = enc.Uint32(i.mmap[pos : pos+offWidth])          // 레코드의 인덱스 오프셋 확인 (in == out) true 여야 됨
	pos = enc.Uint64(i.mmap[pos+offWidth : pos+entWidth]) // 오프셋에 연결된 실제 데이터의 파일에서의 위치
	return out, pos, nil
}

func (i *index) Write(off uint32, pos uint64) error {
	if uint64(len(i.mmap)) < i.size+entWidth { // 먼저 메모리맵 파일에 공간이 있는지 확인
		return io.EOF
	}
	enc.PutUint32(i.mmap[i.size:i.size+offWidth], off)          // 레코드의 인덱스 오프셋 저장
	enc.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos) // 레코드의 실제 저장 위치 저장
	i.size += uint64(entWidth)
	return nil
}

func (i *index) Name() string {
	return i.file.Name()
}
