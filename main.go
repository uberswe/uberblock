package main

import (
	"log"
	"github.com/btcsuite/btcd/rpcclient"
	"fmt"
	"net/http"
	"rsc.io/letsencrypt"
	"html/template"
	"github.com/spf13/viper"
	"github.com/fsnotify/fsnotify"
	"mime"
)

var host = "127.0.0.1:8332"
var user = ""
var pass = ""
var enable_letsencrypt bool
var host_port = "8080"
var theme = "uberblock-light"

var connected     = false
var client *rpcclient.Client

type TemplateData struct {
	Title string
	Connect string
	BlockCount string
	BestBlock string
	Difficulty string
	Chain string
	VerficationProgress string
	Donate string
}

func main() {

	viper.SetDefault("rpc_host", "127.0.0.1:8332")
	viper.SetDefault("rpc_user", "")
	viper.SetDefault("rpc_pass", "")
	viper.SetDefault("host_port", "8080")
	viper.SetDefault("enable_letsencrypt", true)
	viper.SetDefault("theme", "uberblock-light")

	viper.SetConfigName("uberblock")
	viper.AddConfigPath("$HOME/.uberblock")   // path to look for the config file in
	viper.AddConfigPath(".")   // path to look for the config file in

	err := viper.ReadInConfig() // Find and read the config file

	if err != nil { // Handle errors reading the config file
		log.Println("Fatal error config file: %s \n", err)
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file changed:", e.Name)
	})
	host = viper.GetString("rpc_host")
	user = viper.GetString("rpc_user")
	pass = viper.GetString("rpc_pass")
	host_port = viper.GetString("host_port")
	enable_letsencrypt = viper.GetBool("enable_letsencrypt")
	theme = viper.GetString("theme")


	log.Println("v0.1.1")

	connCfg := &rpcclient.ConnConfig{
		Host:         host,
		User:         user,
		Pass:         pass,
		DisableTLS:   true,
		HTTPPostMode: true,
	}
	client, err = rpcclient.New(connCfg, nil)
	if err != nil {
		log.Println(err)
	} else {
		connected = true
	}

	if (connected) {


		log.Printf("Client connected")

		blockCount, err := client.GetBlockCount()
		if err != nil {
			log.Println(err)

			connected = false
		}
		log.Printf("Block count: %d", blockCount)

		chainInfo, err := client.GetBlockChainInfo()
		if err != nil {
			log.Println(err)
			connected = false
		}

		if (connected) {
			log.Printf("Best block: %s\n"+
				"Difficulty: %f\n"+
				"Chain: %s\n"+
				"Verification Progress: %f\n",
				chainInfo.BestBlockHash,
				chainInfo.Difficulty,
				chainInfo.Chain,
				chainInfo.VerificationProgress)
		}
	}

	http.HandleFunc("/assets/", AssetResponse)
	http.HandleFunc("/", UberblockRespond)

	if (enable_letsencrypt) {
		var m letsencrypt.Manager
		if err := m.CacheFile("letsencrypt.cache"); err != nil {
			log.Fatal(err)
		}
		log.Fatal(m.Serve())
	} else {
		err := http.ListenAndServe(":" + host_port, nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}

}

func UberblockRespond(w http.ResponseWriter, r *http.Request) {
	var err error
	if (!connected) {
		if err != nil {
			log.Println(err)
		}

		connCfg := &rpcclient.ConnConfig{
			Host:         host,
			User:         user,
			Pass:         pass,
			DisableTLS:   true,
			HTTPPostMode: true,
		}

		client, err = rpcclient.New(connCfg, nil)
		if err != nil {
			log.Println(err)
			connected = false
		} else {
			connected = true
		}
	}

	if (connected) {

		log.Printf("Client connected")

		chainInfo, err := client.GetBlockChainInfo()
		if err != nil {
			log.Println(err)
			connected = false
		}
		log.Printf("Chain info returned")

		blockCount, err := client.GetBlockCount()
		if err != nil {
			log.Println(err)
			connected = false
		}

		if (connected) {
			td := TemplateData{
				Title:"UberBlock.co Bitcoin Node",
				Connect:"Connect via uberblock.co:8333",
				BlockCount:fmt.Sprintf("Current block count: %d", blockCount),
				BestBlock:fmt.Sprintf("Best block: %s", chainInfo.BestBlockHash),
				Difficulty:fmt.Sprintf("Difficulty: %f", chainInfo.Difficulty),
				Chain:fmt.Sprintf("Chain: %s", chainInfo.Chain),
				VerficationProgress:fmt.Sprintf("Verification Progress: %f", chainInfo.VerificationProgress),
				Donate:fmt.Sprintf("Donate BTC: %s", "bc1qv9ea75xq74mh3jpw2p0puk2vkkkfjqx0rtaw9h"),
			}
			writeToTemplate(w, td)
		} else {

			td := TemplateData{
				Title:"UberBlock.co Bitcoin Node",
				Connect:"Connect via uberblock.co:8333",
				Donate:fmt.Sprintf("Donate BTC: %s", "bc1qv9ea75xq74mh3jpw2p0puk2vkkkfjqx0rtaw9h"),
			}
			writeToTemplate(w, td)
		}
	} else {

		td := TemplateData{
			Title:"UberBlock.co Bitcoin Node",
			Connect:"Connect via uberblock.co:8333",
			Donate:fmt.Sprintf("Donate BTC: %s", "bc1qv9ea75xq74mh3jpw2p0puk2vkkkfjqx0rtaw9h"),
		}
		writeToTemplate(w, td)
	}
}

func writeToTemplate(w http.ResponseWriter, td TemplateData) {
	// TODO check for bind files and template directory as backup
	data, err := Asset("assets/theme/" + theme + "/index.html")
	if err != nil {
		// Asset was not found.
		log.Println("index.html could not be found")
	}
	t, _ := template.New("Index Template").Parse(string(data))
	t.Execute(w, td)                             // merge.

}

func AssetResponse(w http.ResponseWriter, r *http.Request) {

	filepath := "assets/theme/" + theme + r.URL.Path
	contentType, err := GetFileContentType(filepath)
	if err != nil {
		// Asset was not found.
		errorHandler(w, r, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-type", contentType)

	data, err := Asset(filepath)
	if err != nil {
		// Asset was not found.
		errorHandler(w, r, http.StatusNotFound)
		return
	}
	w.Write(data)

	// use asset data
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprint(w, "404 Not Found")
	}
}

func GetFileContentType(path string) (string, error) {

	contentType := mime.TypeByExtension(path)
	//f, err := os.Open(path)
	//if err != nil {
	//	panic(err)
	//}
	//defer f.Close()
	//
	//// Only the first 512 bytes are used to sniff the content type.
	//buffer := make([]byte, 512)
	//
	//_, err = f.Read(buffer)
	//if err != nil {
	//	return "", err
	//}
	//
	//// Use the net/http package's handy DectectContentType function. Always returns a valid
	//// content-type by returning "application/octet-stream" if no others seemed to match.
	//contentType := http.DetectContentType(buffer)

	return contentType, nil
}