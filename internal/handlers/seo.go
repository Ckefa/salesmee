package handlers

import (
	"encoding/xml"
	"net/http"

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
	c.XML(http.StatusOK, urlset{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: []url{
			{Loc: "/", Priority: "1.0"},
			{Loc: "/guide", Priority: "0.9"},
			{Loc: "/privacy", Priority: "0.5"},
			{Loc: "/terms", Priority: "0.5"},
			{Loc: "/cookies", Priority: "0.4"},
			{Loc: "/user-deletion", Priority: "0.3"},
			{Loc: "/business/login", Priority: "0.6"},
			{Loc: "/business/register", Priority: "0.8"},
		},
	})
}

func RobotsTXT(c *gin.Context) {
	c.String(http.StatusOK, `User-agent: *
Allow: /
Disallow: /business/
Disallow: /client/
Disallow: /admin/
Disallow: /api/

Sitemap: https://`+c.Request.Host+`/sitemap.xml
`)
}
