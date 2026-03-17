package pagetext

import (
	"fmt"

	"agentium/internal/model"
	"agentium/internal/session"
)

const extractScript = `
() => {
  const source = document.body || document.documentElement;
  const text = (source?.innerText || source?.textContent || '')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 20000);

  return {
    url: location.href,
    title: document.title,
    text
  };
}
`

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Capture(runtime *session.Runtime) (model.PageText, error) {
	result, err := runtime.Page.Eval(extractScript)
	if err != nil {
		return model.PageText{}, fmt.Errorf("capture page text: %w", err)
	}

	var pageText model.PageText
	if err := result.Value.Unmarshal(&pageText); err != nil {
		return model.PageText{}, fmt.Errorf("decode page text: %w", err)
	}

	return pageText, nil
}
