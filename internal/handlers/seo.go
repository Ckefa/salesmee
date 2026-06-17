package handlers

import (
	"encoding/xml"
	"net/http"

	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
)

type urlset struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []url    `xml:"url"`
}

type url struct {
	Loc      string `xml:"loc"`
	Priority string `xml:"priority"`
}

func SitemapXML(c *gin.Context) {
	base := services.GetBaseURL(c)
	c.XML(http.StatusOK, urlset{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: []url{
			{Loc: base + "/", Priority: "1.0"},
			{Loc: base + "/guide", Priority: "0.9"},
			{Loc: base + "/privacy", Priority: "0.5"},
			{Loc: base + "/terms", Priority: "0.5"},
			{Loc: base + "/cookies", Priority: "0.4"},
			{Loc: base + "/user-deletion", Priority: "0.3"},
			{Loc: base + "/business/login", Priority: "0.6"},
			{Loc: base + "/business/register", Priority: "0.8"},
			{Loc: base + "/client/login", Priority: "0.6"},
		},
	})
}

func RobotsTXT(c *gin.Context) {
	c.String(http.StatusOK, `User-agent: *
Allow: /
Allow: /client/login$
Allow: /business/login$
Allow: /business/register$
Disallow: /business/
Disallow: /client/
Disallow: /admin/
Disallow: /api/

Sitemap: `+services.GetBaseURL(c)+`/sitemap.xml
`)
}
