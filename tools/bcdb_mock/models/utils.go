package models

import (
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// DefaultPageSize is default page size
	DefaultPageSize int64 = 100
)

// Pager is the find options wrapper
type Pager *options.FindOptions

// Page creates a Pager
func Page(p int64, size ...int64) Pager {
	ps := DefaultPageSize
	if len(size) > 0 {
		ps = size[0]
	}
	skip := p * ps
	return options.Find().SetLimit(ps).SetSkip(skip).SetSort(bson.D{{Key: "_id", Value: -1}})
}

// PageFromRequest return page information from the page & size url params
func PageFromRequest(r *http.Request) Pager {
	var (
		p = r.FormValue("page")
		s = r.FormValue("size")

		page int64
		size = DefaultPageSize
	)

	if len(p) != 0 {
		page, _ = strconv.ParseInt(p, 10, 64)
		// user facing pages are start from 1
		page--
		if page < 0 {
			page = 0
		}
	}
	if len(s) != 0 {
		size, _ = strconv.ParseInt(s, 10, 64)
	}

	// make sure user doesn't kill the server by returning too much data
	if size > 1000 {
		size = 1000
	}

	return Page(page, size)
}
