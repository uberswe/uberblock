Uberblock
=========
Currently uberblock is a simple web page for Bitcoin but
my plan is to make a general block explorer that can be configured
to work with many different cryptocurrencies.

Configuration
=============

go-bindata is used to generate the bindata.go file and needs 
to be run every time you change any of the themes. To build the
uberblock binary I usually use the following command:

```
go-bindata assets/... && go build
```

You can build for a specific os like so

```
env GOOS=linux GOARCH=amd64 go build 
```

You can create a uberblock.json file in the same directory as
the binary or in the root/home folder on your server. The following
are possible parameters.

- rpc_user - Bitcoin node RPC user
- rpc_pass - Bitcoin node RPC pass
- rpc_host - ip/domain and port for your Bitcoin full node, works with bitcoind and btcd (not fully tested)
- enable_letsencrypt - Enables letsencrypt and tries to serve on 443, domain should be configured before this is run
- host_port - Port where the web page will be served, Ignored if letsencrypt is used
- theme - Set to the folder name of the theme

Contribute
==========
I am open to contributions, feel free to make a fork 
and pull request. I will try to review all pull requests
as quickly as possible. If you have any questions feel free
to send me an email.

License
=======

The code is provided under the MIT License

Donate
======

If you find my code useful feel free to donate

Donate BTC: 15cNb7zvWJDbZwV4xHvGyV96AtcyqUSkSH
Donate ETH: 0xB9Df510bE5Aaad76E558cc7BF41E6363f3944dfc
Donate LTC: LUQBfTqkeS4UuoF9nxsEqAVysb2ibUzbZi