package promotion

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"strings"
)

type Service struct {
	promotions []template.HTML
}

func New() *Service {
	s := Service{}
	items, _ := os.ReadDir("./template/dist/include/promotional/")
	for _, item := range items {
		if !item.IsDir() && strings.HasSuffix(item.Name(), ".html") {
			// Service is non critical, ignore errors
			t, _ := template.ParseFiles(fmt.Sprintf("./template/dist/include/promotional/%s", item.Name()))
			var tpl bytes.Buffer
			_ = t.Execute(&tpl, nil)
			s.promotions = append(s.promotions, template.HTML(tpl.String()))
		}
	}
	return &s
}

func (s *Service) RandomPromotion() template.HTML {
	return s.promotions[rand.Intn(len(s.promotions))]
}
