# mapper
Allows for easily creating maps for spyglass 2

## Getting up and running
1. you must have a working Go installation [https://golang.org/](https://golang.org/)
2. clone this repo
3. change to the repo directory
4. run `go get github.com/anaskhan96/soup`
5. from the root directory of the repo, run `go generate` then `go build`


## Using the tool
1. modify the maps in the `maps/` subdirectory (prepopulated with dotlan maps)
2. run the `spyglass_mapper` binary (no feedback will be given)
3. open a web browser to `http://localhost:8334`
4. browse to the relevant map listed on the page.
5. each time you reload the browser the map will be reread from disk and show changes (systems will be either red or white randomly)
