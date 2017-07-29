# kanali-plugin-apikey

[![Travis branch](https://img.shields.io/travis/northwesternmutual/kanali-plugin-apikey/master.svg?style=flat-square)](https://travis-ci.org/northwesternmutual/kanali-plugin-apikey) [![Coveralls branch](https://img.shields.io/coveralls/northwesternmutual/kanali-plugin-apikey/master.svg?style=flat-square)](https://coveralls.io/github/northwesternmutual/kanali-plugin-apikey) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/northwesternmutual/kanali-plugin-apikey)

> a plugin for Kanali

# Local Development

Below are the steps to follow if you want to build/test locally. [Glide](https://glide.sh/) is a dependency.

```sh
$ mkdir -p $GOPATH/src/github.com/northwesternmutual
$ cd $GOPATH/src/github.com/northwesternmutual
$ git clone https://github.com/northwesternmutual/kanali-plugin-apikey
$ cd kanali-plugin-apikey
$ make install_ci
$ make kanali-plugin-apikey
```