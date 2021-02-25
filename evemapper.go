package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	svg "github.com/ajstarks/svgo"
)

type (
	EveMapper struct{
		Galaxy NewEden
	}

	spyglassMap struct {
		Name        string `json:"name"`
		Author      string `json:"author,omitempty"`
		Description string `json:"description,omitempty"`

		Systems map[int32]spyglassSystem `json:"systems"`
		Width   int32                    `json:"width"`
		Height  int32                    `json:"height"`
	}

	spyglassSystem struct {
		ID   int32  `json:"id"`
		Name string `json:"name"`
		Icon string `json:"icon,omitempty"`
		X    int32  `json:"x"`
		Y    int32  `json:"y"`
		External bool `json:"external,omitempty"`
	}
)

func NewEveMapper() *EveMapper {

	g := make(NewEden)
	err  := g.LoadData()
	if err != nil {
		log.Fatal(err)
	}



	return &EveMapper{
		Galaxy: g,
	}
}

func (em *EveMapper) ListenAndServe() error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.NoCache)

	r.Get("/", em.viewIndex)
	r.Route("/map", func(r chi.Router) {
		r.Get("/{map}", em.viewMap)
	})

	return http.ListenAndServe(":8334", r)
}

func (em *EveMapper) viewIndex(w http.ResponseWriter, r *http.Request) {

	root, err := filepath.Abs("./maps")
	if err != nil{
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	var files []string

	err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		name := info.Name()
		ext := filepath.Ext(name)
		name = strings.TrimSuffix(name, ext)
		if name == "maps" {
			return nil
		}
		files = append(files, name)
		return nil
	})

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(200)

	for _, f := range files {
		w.Write([]byte(fmt.Sprintf("<a href=\"/map/%s\">%s</a><br />\n", f, f)))
	}

}

func (em *EveMapper) viewMap(w http.ResponseWriter, r *http.Request) {
	mapid := chi.URLParam(r, "map")
	p, err := filepath.Abs("./maps/" + mapid + ".json")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	f, err := os.Open(p)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	var m spyglassMap

	err = dec.Decode(&m)
	if err != nil {
		w.WriteHeader(406)
		w.Write([]byte(err.Error()))
		return
	}

	out, err := em.CreateMapSVG(m)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	fmt.Fprint(w, out)

}

func (em *EveMapper) CreateMapSVG(mp spyglassMap) (string, error){
	start := time.Now()

	const systemWidth = 50
	const systemHeight = 22
	const systemRounded = 10

	systems := make([]int32, len(mp.Systems))
	for _, s := range mp.Systems{
		systems = append(systems, s.ID)
	}

	// TODO, allow custom connections
	var connections []string


	var buf bytes.Buffer

	canvas := svg.New(&buf)
	canvas.Start(int(mp.Width), int(mp.Height))

	//Draw a border
	canvas.Rect(0,0,int(mp.Width), int(mp.Height), "fill:rgb(255,255,255);stroke:rgb(0,0,0);stroke-width:1px")

	// First draw all of the connections so that they are beneath all other things. Keep them in their own group

	connections = append(connections, em.GetJumps(systems)...)

	canvas.Gid("jumps")
	for _, con := range connections {
		sp := strings.Split(con, "-")
		if len(sp) != 2 {
			continue
		}

		source, sok := strconv.Atoi(sp[0])
		dest, dok := strconv.Atoi(sp[1])
		if ((sok != nil) || (dok != nil)) {
			log.Println(sp[0], sp[1])
			log.Println("Not ints")
			continue
		}

		src, srok := mp.Systems[int32(source)]
		dst, dtok := mp.Systems[int32(dest)]
		if !(srok || dtok) {
			log.Println(con)
			log.Println("not present")
			continue
		}
		// Get middle point of source system
		startX := src.X + (systemWidth / 2)
		startY := src.Y + (systemHeight / 2)

		//	Get middle point of destination system
		endX := dst.X + (systemWidth / 2)
		endY := dst.Y + (systemHeight / 2)

		// TODO implement line colours
		// TODO investigate use of beziers
		canvas.Line(int(startX), int(startY), int(endX), int(endY), "stroke:rgb(0,0,0);stroke-width:1px")
	}
	canvas.Gend()

	//	Now add all of the systems to the map
	// Each system is a rounded rect with a height of 30, width of 62, r of 10
	canvas.Gid("systems")
	for _, s := range mp.Systems {
		// Start an individual group for each system
		canvas.Gid(strconv.Itoa(int(s.ID)))
		status := rand.Float32() > 0.5
		style := "fill:rgb(255,255,255);stroke:rgb(0,0,0);stroke-width:1px"
		if status {
			style = "fill:rgb(255,64,64);stroke:rgb(0,0,0);stroke-width:1px"
		}


		rnd := systemRounded
		if s.External {
			rnd = 0
		}

		canvas.Roundrect(int(s.X), int(s.Y), systemWidth, systemHeight, systemRounded, rnd, style)

		//	create the system name text
		name := s.Name
		stat := "STATUS!"
		x := s.X + (systemWidth / 2)
		yn := s.Y + (systemHeight / 2)
		ys := s.Y + (systemHeight * 7 / 8)

		canvas.Text(int(x), int(yn), name, "text-anchor:middle;font-size:9px")
		canvas.Text(int(x), int(ys), stat, "text-anchor:middle;font-size:8px")
		canvas.Gend()
	}

	canvas.Gend()

	canvas.End()

	log.Printf("Generation took %v", time.Since(start))

	return buf.String(), nil
}

// GetJumps will return the connections between the monitored systems
// This list will contain both directions ie 1 -> 2 and 2 -> 1
func (em *EveMapper) GetJumps(systems []int32) []string {
	// TODO find a way to preallocate this to some extent
	jumps := make([]string, 0)

	for _, s := range systems {
		source, err := em.Galaxy.GetSystem(s)
		if err != nil {
			// TODO this shouldnt ever happen so I probably shouldnt be silent here but will do for now
			continue
		}

		for _, gate := range source.Stargates {
			if em.IsSystemMonitored(gate.Destination.SystemID, systems) {
				jumps = append(jumps, strconv.Itoa(int(source.SystemID))+"-"+strconv.Itoa(int(gate.Destination.SystemID)))
			}
		}
	}
	return jumps
}

func (em *EveMapper) IsSystemMonitored(sys int32, syss []int32) bool {
	for _, s := range syss {
		if s == sys {
			return true
		}
	}
	return false
}



