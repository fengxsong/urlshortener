# urlshortener demo.

use memory to store urls, so it can only be run stand-alone.

autoload objects from file and dump to file when system interrupt.

comes with `gin` web framework.

Usage:
generate short url:

	curl -X POST -d '{"orig": "https://fengxsong.github.io"}' http://localhost:8000/v1/
	{"Short":"jxtjX","Orig":"https://fengxsong.github.io","Create":"2016-12-22T11:43:57.665721+08:00","Click":0,"Expiration":"2016-12-22T11:48:57.665721+08:00"}

get short url stats:

	curl -X GET http://localhost:8000/v1/jxtjX?stats=xxx

redirect to original url

    curl -X GET http://localhost:8000/v1/jxtjX

q: should I reduce the interval of removing the expired keys?

