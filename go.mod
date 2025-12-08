module github.com/tstromberg/gocachemark

go 1.25.4

require (
	github.com/Code-Hex/go-generics-cache v1.3.1
	github.com/Yiling-J/theine-go v0.6.0
	github.com/codeGROOVE-dev/sfcache v1.1.2
	github.com/coocood/freecache v1.2.4
	github.com/dgraph-io/ristretto v0.2.0
	github.com/dgryski/go-s4lru v0.0.0-20150401095600-fd9b33c61bfe
	github.com/elastic/go-freelru v0.16.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/klauspost/compress v1.18.0
	github.com/maypok86/otter/v2 v2.2.1
	github.com/scalalang2/golang-fifo v1.2.0
	github.com/vmihailenco/go-tinylfu v0.2.2
	github.com/zeebo/xxh3 v1.0.2
)

replace github.com/codeGROOVE-dev/sfcache => /Users/t/dev/r2r/sfcache

replace github.com/codeGROOVE-dev/sfcache/pkg/persist => /Users/t/dev/r2r/sfcache/pkg/persist

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/codeGROOVE-dev/sfcache/pkg/persist v1.1.4 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/sys v0.34.0 // indirect
)
