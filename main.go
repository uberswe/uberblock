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
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"path"
	"strconv"
)

var host = "uberblock.co:8332"
var user = ""
var pass = ""
var enable_letsencrypt bool
var host_port = "8080"
var theme = "uberblock-light"

var connected     = false
var client *rpcclient.Client

type TemplateData struct {
	Title string
	BlockChainInfo map[string]string
	Donate string
}

type Setting struct {
	gorm.Model
	CurrentBlock uint
}

func main() {

	viper.SetDefault("rpc_host", "uberblock.co:8332")
	viper.SetDefault("rpc_user", "")
	viper.SetDefault("rpc_pass", "")
	viper.SetDefault("host_port", "8080")
	viper.SetDefault("enable_letsencrypt", false)
	viper.SetDefault("theme", "uberblock-light")
	viper.SetDefault("db_path", "")

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
	db_path := viper.GetString("db_path")
	db, err := gorm.Open("sqlite3", path.Join(db_path, "database.db"))
	if err != nil {
		panic("failed to connect to database: " + path.Join(db_path, "database.db"))
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&Setting{})
	db.AutoMigrate(&Transaction{})
	db.AutoMigrate(&Address{})

	log.Println("Migrated database")

	log.Println("v0.2.0")

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

	//UberblockParse()

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

/*
 * Parses all blocks and builds database of transactions
 */

func UberblockParse() {
	fmt.Println("Begin parsing blockchain...")
	count := 0

	db_path := viper.GetString("db_path")
	db, err := gorm.Open("sqlite3", path.Join(db_path, "database.db"))
	if err != nil {
		panic("failed to connect to database: " + path.Join(db_path, "database.db"))
	}
	defer db.Close()

	var setting Setting

	db.First(&setting, 1)

	if setting.ID == 0 {
		fmt.Println("No settings, starting from first block")
		db.Create(&Setting{CurrentBlock:0})
		db.First(&setting, 1)
	}


	fmt.Println("Starting at block " + strconv.FormatInt(int64(setting.CurrentBlock), 10))

	blockcount, err := client.GetBlockCount()
	if err != nil {
		log.Println(err)
		connected = false
	}

	for blockcount > int64(setting.CurrentBlock) {
		count = 0

		blockhash, err := client.GetBlockHash(int64(setting.CurrentBlock))
		if err != nil {
			log.Println(err)
			connected = false
		}
		block, err := client.GetBlock(blockhash)
		if err != nil {
			log.Println(err)
			connected = false
		}
		for _, t := range block.Transactions {
			txhash := t.TxHash()
			txnOut, err := client.GetTxOut(&txhash, 0, false)
			if err != nil {
				log.Println(err)
				connected = false
			}
			txnIn, err := client.GetRawTransaction(&txhash)
			if err != nil {
				log.Println(err)
				connected = false
			}
			fmt.Println(txnIn.Hash().String())
			if txnOut != nil {

				var transaction Transaction

				db.First(&transaction, "Hash = ?", txhash.String())

				if transaction.ID == 0 {
					db.Create(&Transaction{
						Hash:txhash.String(),
						Version:uint(txnOut.Version),
						Blockhash:blockhash.String(),
					})
					for _, addr := range txnOut.ScriptPubKey.Addresses {
						var address Address

						db.First(&address, "Hash = ?", addr)
						if address.ID == 0 {
							db.Create(&Address{
								Hash:    addr,
								Balance: txnOut.Value,
							})
							db.First(&address, "Hash = ?", addr)
						} else {
							balance := address.Balance + txnOut.Value
							db.Model(&address).Update("Balance", balance)
						}
						db.First(&address, "Hash = ?", addr)
						db.First(&transaction, "Hash = ?", txhash.String())
						db.Model(&address).Association("Transactions").Append(&transaction)
					}
				}

			}
			count++
		}
		if setting.CurrentBlock % 100 == 0 {
			println("Transactions: " + strconv.FormatInt(int64(count), 10) + " Block: " + strconv.FormatInt(int64(setting.CurrentBlock), 10) + " " + blockhash.String())
		}
		setting.CurrentBlock++
		db.Model(&setting).Update("CurrentBlock", setting.CurrentBlock)
	}
}

func FloatToString(input_num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 12, 64)
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

	bcinfo := make(map[string]string)

	bcinfo["Connect"] = "Connect via uberblock.co:8333"

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

			bcinfo["BlockCount"] = fmt.Sprintf("Current block count: %d", blockCount)
			bcinfo["BestBlock"] = fmt.Sprintf("Best block: %s", chainInfo.BestBlockHash)
			bcinfo["Difficulty"] = fmt.Sprintf("Difficulty: %f", chainInfo.Difficulty)
			bcinfo["Chain"] = fmt.Sprintf("Chain: %s", chainInfo.Chain)
			bcinfo["VerficationProgress"] = fmt.Sprintf("Verification Progress: %f%", chainInfo.VerificationProgress)

			td := TemplateData{
				Title:"UberBlock.co Bitcoin Node",
				BlockChainInfo:bcinfo,
				Donate:fmt.Sprintf("Donate BTC: %s", "bc1qv9ea75xq74mh3jpw2p0puk2vkkkfjqx0rtaw9h"),
			}
			writeToTemplate(w, td)
		} else {

			bcinfo["Maintenance"] = "Currently under maintenance, email m@rkus.io for more information"

			td := TemplateData{
				Title:"UberBlock.co Bitcoin Node",
				BlockChainInfo:bcinfo,
				Donate:fmt.Sprintf("Donate BTC: %s", "bc1qv9ea75xq74mh3jpw2p0puk2vkkkfjqx0rtaw9h"),
			}
			writeToTemplate(w, td)
		}
	} else {

		bcinfo["Maintenance"] = "Currently under maintenance, email m@rkus.io for more information"

		td := TemplateData{
			Title:"UberBlock.co Bitcoin Node",
			BlockChainInfo:bcinfo,
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
	return contentType, nil
}