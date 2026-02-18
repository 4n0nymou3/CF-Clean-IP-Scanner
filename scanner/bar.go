package scanner

import (
	"fmt"

	"github.com/cheggaaa/pb/v3"
)

type Bar struct {
	bar *pb.ProgressBar
}

func newBar(count int, myStrStart, myStrEnd string) *Bar {
	tmpl := fmt.Sprintf(`{{counters . }} {{ bar . "[" "-" (cycle . "↖" "↗" "↘" "↙" ) "_" "]"}} %s {{string . "MyStr" | green}} %s {{rtime . | blue}}`, myStrStart, myStrEnd)
	b := pb.ProgressBarTemplate(tmpl).Start(count)
	return &Bar{bar: b}
}

func (b *Bar) grow(num int, myStrVal string) {
	b.bar.Set("MyStr", myStrVal).Add(num)
}

func (b *Bar) done() {
	b.bar.Finish()
}