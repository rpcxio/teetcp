package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *TCPServer) startAdminServer(addr string) {
	router := httprouter.New()
	router.GET("/startdw/:id", s.startDW)
	router.GET("/stopdw", s.stopDW)
	router.GET("/dw", s.getDW)

	http.ListenAndServe(addr, router)
}

func (s *TCPServer) startDW(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	if s.s2 != "" {
		http.Error(w, "双写已开启:"+s.s2, http.StatusInternalServerError)
	}
	s.s2 = id
	close(s.s2ch)
}

func (s *TCPServer) stopDW(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	s.s2 = ""
	s.s2ch = make(chan struct{})
}

func (s *TCPServer) getDW(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte(s.s2))
}
