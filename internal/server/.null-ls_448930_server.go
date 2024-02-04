package server

import "net/http"

func Start(ip, port string) error {
	mux := http.NewServeMux()

	if err := http.ListenAndServe(ip+port, mux); err != nil {
		return err
	}

	return nil
}
