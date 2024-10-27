# extract url path

```
curl -i -X GET http://localhost:4221/

curl -i -X GET http://localhost:4221/not-exists

curl -i -X GET http://localhost:4221/echo/value-to-echo

curl -i -X GET http://localhost:4221/files/test.txt

curl -v --data "12345" -H "Content-Type: application/octet-stream" http://localhost:4221/files/file_123

curl -v -H "Accept-Encoding: gzip" http://localhost:4221/echo/abc

curl -v -H "Accept-Encoding: gzip" http://localhost:4221/echo/abc | gunzip
```
