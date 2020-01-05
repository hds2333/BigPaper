module bigpaper

go 1.13

require (
	github.com/dennis/http v0.0.0
	github.com/gin-gonic/gin v1.5.0
	github.com/go-redis/redis v6.15.6+incompatible
	utils v0.0.0
)

replace github.com/dennis/http v0.0.0 => ./dennis/http

replace utils v0.0.0 => ./utils
