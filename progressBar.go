package progressBar

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gitlab.com/xyzyx760326/golanglib/rewrite"
)

type finishStatus_t int

const (
	haveErr finishStatus_t = iota
	normal
	forceStop
	inProgress
)

type Progress_t struct {
	name         string
	isSingleMode bool
	isSingleLine bool
	max          int
	current      int
	msg          string
	finish       finishStatus_t
	err          error
	statCh       chan Progress_t
}

type ProgressBarManager_t struct {
	barCount int

	timeOut time.Duration
	statCh  chan Progress_t
}

const (
	left      = "["
	right     = "]"
	unLoad    = " "
	loaded    = "#"
	width     = 40
	msgLength = 40
)

func CreateBarManager(timeOut time.Duration) *ProgressBarManager_t {
	return &ProgressBarManager_t{
		statCh:  make(chan Progress_t, 100),
		timeOut: timeOut,
	}
}

func CreateSingleBar(length int, name string, isSingleLine bool) *Progress_t {
	pb := Progress_t{
		name:         name,
		isSingleMode: true,
		isSingleLine: isSingleLine,
		max:          length,
		current:      0,
		msg:          "",
		finish:       inProgress,
	}

	return &pb
}

func (p *ProgressBarManager_t) Create(length int, name string) *Progress_t {
	pb := Progress_t{
		name:         name,
		isSingleMode: false,
		max:          length,
		current:      0,
		msg:          "",
		statCh:       p.statCh,
		finish:       inProgress,
	}

	p.barCount++

	return &pb
}

func (p *ProgressBarManager_t) ShowAndWait() []string {
	var barSummary = map[string]Progress_t{}
	var barOrder = map[int]string{}
	index := 0

	timeOutTimer := time.NewTimer(p.timeOut)
	rw := rewrite.Create()

LOOP:
	for {
		select {
		case barInfo := <-p.statCh:
			_, ok := barSummary[barInfo.name]
			if !ok {
				barOrder[index] = barInfo.name
				index++
			}

			barSummary[barInfo.name] = barInfo

			isFinishedCount := 0
			lines := []string{}
			for i := 0; i < len(barOrder); i++ {
				showStr := ""
				if len(barOrder[i]) >= 7 {
					showStr = barOrder[i][:7] + "..."
				} else {
					showStr = barOrder[i]
				}
				lines = append(lines, fmt.Sprintf("  %s : %s", showStr, barSummary[barOrder[i]].msg))
				lines = append(lines, fmt.Sprintf("    %s", barStatus(barSummary[barOrder[i]].current, barSummary[barOrder[i]].max)))
				if barSummary[barOrder[i]].finish != inProgress {
					isFinishedCount++
				}
			}

			rw.PrintMultiln(lines)
			if isFinishedCount == p.barCount {
				break LOOP
			} else {
				rw.MoveCursorBack()
				timeOutTimer.Reset(p.timeOut)
			}
			break
		case <-timeOutTimer.C:
			rw.Stop()
			break LOOP
		}
	}

	unFinishBar := []string{}
	for _, bar := range barSummary {
		if bar.finish != normal {
			unFinishBar = append(unFinishBar, bar.name)
		}
	}

	close(p.statCh)

	if len(unFinishBar) != 0 {
		str := "Process not finished: \n  " + strings.Join(unFinishBar, "\n  ")
		fmt.Println(str)
	}

	return unFinishBar
}

func (p *Progress_t) Increment(length int, msg string) {
	if p.finish != normal && p.finish != inProgress {
		return
	}

	p.current = p.current + length

	msg = strings.Replace(msg, "\r\n", " ", -1)
	msg = strings.Replace(msg, "\n", " ", -1)

	if len(msg) > msgLength {
		msg = msg[len(msg)-msgLength:]
	} else {
		tempSpace := ""
		for i := len(msg); i < msgLength; i++ {
			tempSpace = tempSpace + " "
		}
		msg = msg + tempSpace
	}

	p.msg = msg

	if p.current > p.max {
		p.err = errors.New("Out of range")
		p.finish = haveErr
	} else if p.current == p.max {
		p.finish = normal
	}

	if p.isSingleMode {
		p.Show()
	} else {
		p.statCh <- *p
	}
}

func (p *Progress_t) IsFinished() bool {
	if p.finish == normal {
		return true
	} else {
		return false
	}
}

func (p *Progress_t) ForceStop(err error) {
	p.finish = forceStop
	if err != nil {
		p.err = err
	}

	go func() {
		p.statCh <- *p
	}()
}

func (p *Progress_t) Show() {
	if p.isSingleLine {
		fmt.Printf("\r %s %s %s", p.name, barStatus(p.current, p.max), p.msg)
		if p.finish != inProgress {
			fmt.Println()
		}
	} else {
		lines := []string{}
		lines = append(lines, fmt.Sprintf("  %s : %s", p.name, p.msg))
		lines = append(lines, fmt.Sprintf("    %s", barStatus(p.current, p.max)))
		rw := rewrite.Create()
		rw.PrintMultiln(lines)
		if p.finish == inProgress {
			rw.MoveCursorBack()
		}
	}
}

func barStatus(current int, max int) string {

	var loadedCount int
	loadedCount = (current * width / max)

	var printStr string
	printStr += left

	for i := 0; i < width; i++ {
		if i < loadedCount {
			printStr += loaded
		} else {
			printStr += unLoad
		}
	}

	printStr += right
	printStr = fmt.Sprintf("%s %02d%%", printStr, current*100/max)

	return printStr
}
