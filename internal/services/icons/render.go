package icons

import (
	"html/template"
	"strings"
)

func injectStroke(svg string, cls string) string {
	if !strings.Contains(svg, `stroke="currentColor"`) {
		svg = strings.Replace(svg, `fill="none"`, `fill="none" stroke="currentColor"`, 1)
	}
	if cls != "" {
		svg = strings.Replace(svg, "<svg", `<svg class="`+cls+`" width="1.25em" height="1.25em"`, 1)
	} else {
		svg = strings.Replace(svg, "<svg", `<svg width="1.25em" height="1.25em"`, 1)
	}
	return svg
}

func Heroicon(name string, class ...string) template.HTML {
	svg, ok := Icons[name]
	if !ok {
		return template.HTML("")
	}
	cls := strings.Join(class, " ")
	svg = injectStroke(svg, cls)
	return template.HTML(svg)
}

func HeroiconSolid(name string, class ...string) template.HTML {
	solidName := name
	svg, ok := Icons[solidName]
	if !ok {
		return template.HTML("")
	}
	cls := strings.Join(class, " ")
	svg = injectStroke(svg, cls)
	return template.HTML(svg)
}
