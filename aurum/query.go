package aurum

type QueryOption func(*queryOpts)
type queryOpts struct {
	limit, offset int
	orderBy       string
	preload       []string
}

func Limit(limit int) QueryOption      { return func(o *queryOpts) { o.limit = limit } }
func Offset(offset int) QueryOption    { return func(o *queryOpts) { o.offset = offset } }
func OrderBy(order string) QueryOption { return func(o *queryOpts) { o.orderBy = order } }
