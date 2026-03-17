package action

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"agentium/internal/humanize"
	"agentium/internal/model"
	"agentium/internal/session"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

type Service struct {
	mousePath *humanize.MousePathBuilder
	typing    *humanize.TypingDelays
	rng       *rand.Rand
}

func NewService(seed int64) *Service {
	return &Service{
		mousePath: humanize.NewMousePathBuilder(seed),
		typing:    humanize.NewTypingDelays(seed + 1),
		rng:       rand.New(rand.NewSource(seed + 2)),
	}
}

func (s *Service) Execute(runtime *session.Runtime, input model.ActionRequest) (model.ActionResult, error) {
	start := time.Now()
	err := runtime.WithLock(func() error {
		switch input.Action {
		case model.ActionNavigate:
			return runtime.Page.Navigate(*input.Value)
		case model.ActionWaitNetworkIdle:
			return s.waitNetworkIdle(runtime, 10*time.Second)
		case model.ActionClick:
			return s.click(runtime, *input.RefID)
		case model.ActionFill:
			return s.fill(runtime, *input.RefID, *input.Value)
		case model.ActionTypeText:
			return s.typeText(runtime, *input.RefID, *input.Value, input.Options.DelayMS)
		case model.ActionScroll:
			return s.scrollTo(runtime, *input.RefID, input.Options.DelayMS)
		default:
			return errors.New("unsupported action")
		}
	})

	events := runtime.Tracker.Since(start)
	if err != nil {
		message := err.Error()
		return model.ActionResult{
			Success:       false,
			ErrorMsg:      &message,
			NetworkEvents: events,
		}, nil
	}

	return model.ActionResult{
		Success:       true,
		NetworkEvents: events,
	}, nil
}

func (s *Service) click(runtime *session.Runtime, refID int) error {
	element, err := runtime.Page.Element(fmt.Sprintf(`[agentium-id="%d"]`, refID))
	if err != nil {
		return fmt.Errorf("find element: %w", err)
	}

	target, ok := runtime.Ref(refID)
	if !ok {
		return errors.New("ref_id is stale, request a new snapshot")
	}

	if err := element.ScrollIntoView(); err != nil {
		return fmt.Errorf("scroll into view: %w", err)
	}

	if err := s.moveMouse(runtime, target.BBox.X+target.BBox.W/2, target.BBox.Y+target.BBox.H/2); err != nil {
		return err
	}

	if err := runtime.Page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("mouse click: %w", err)
	}

	return s.waitNetworkIdle(runtime, 5*time.Second)
}

func (s *Service) fill(runtime *session.Runtime, refID int, value string) error {
	element, err := runtime.Page.Element(fmt.Sprintf(`[agentium-id="%d"]`, refID))
	if err != nil {
		return fmt.Errorf("find element: %w", err)
	}

	if err := element.Focus(); err != nil {
		return fmt.Errorf("focus element: %w", err)
	}

	_ = element.SelectAllText()

	if err := element.Input(value); err != nil {
		return fmt.Errorf("fill element: %w", err)
	}

	return s.waitNetworkIdle(runtime, 3*time.Second)
}

func (s *Service) typeText(runtime *session.Runtime, refID int, value string, delayMS int) error {
	element, err := runtime.Page.Element(fmt.Sprintf(`[agentium-id="%d"]`, refID))
	if err != nil {
		return fmt.Errorf("find element: %w", err)
	}

	if err := element.Focus(); err != nil {
		return fmt.Errorf("focus element: %w", err)
	}

	baseDelay := 100 * time.Millisecond
	if delayMS > 0 {
		baseDelay = time.Duration(delayMS) * time.Millisecond
	}

	for _, char := range value {
		if err := runtime.Page.Keyboard.Type(input.Key(char)); err != nil {
			return fmt.Errorf("type text: %w", err)
		}
		time.Sleep(s.typing.Next(baseDelay))
	}

	return s.waitNetworkIdle(runtime, 3*time.Second)
}

func (s *Service) scrollTo(runtime *session.Runtime, refID int, delayMS int) error {
	target, ok := runtime.Ref(refID)
	if !ok {
		return errors.New("ref_id is stale, request a new snapshot")
	}

	total := int(target.BBox.Y - 120)
	if total < 0 {
		total = 0
	}

	steps := 6 + s.rng.Intn(6)
	chunk := float64(total) / float64(steps)
	pause := 80 * time.Millisecond
	if delayMS > 0 {
		pause = time.Duration(delayMS) * time.Millisecond
	}

	for i := 0; i < steps; i++ {
		if err := runtime.Page.Mouse.Scroll(0, chunk, 1); err != nil {
			return fmt.Errorf("scroll page: %w", err)
		}
		time.Sleep(pause)
	}

	return s.waitNetworkIdle(runtime, 2*time.Second)
}

func (s *Service) moveMouse(runtime *session.Runtime, x, y float64) error {
	fromX, fromY := runtime.MousePosition()
	for _, point := range s.mousePath.Build(humanize.Point{X: fromX, Y: fromY}, humanize.Point{X: x, Y: y}) {
		if err := runtime.Page.Mouse.MoveTo(proto.Point{X: point.X, Y: point.Y}); err != nil {
			return fmt.Errorf("move mouse: %w", err)
		}
		time.Sleep(12 * time.Millisecond)
	}
	runtime.SetMousePosition(x, y)
	return nil
}

func (s *Service) waitNetworkIdle(runtime *session.Runtime, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if time.Since(runtime.Tracker.LastActivity()) > 500*time.Millisecond {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("network did not become idle before timeout")
}
