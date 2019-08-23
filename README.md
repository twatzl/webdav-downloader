# webdav-downloader

This is a utility to **recursively** download a directory on a webdav server.
I was looking for such a tool, but could not find any existing tool that could do the recursive part, so I decided to write my own.
It can exactly do that and nothing more.

## Usage

1. compile it
2. put the executable to some location which is in your `$PATH`
3. create a config file based on the example provided in this repo
4. `cd` to the folder where you want to download to
5. `webdav-downloader --config ./path/to/config.yaml --directory="/path/to/dir/on/server"`

## Issues

You are welcome to ask questions in the issues page of this repo.

Regarding feature requests: This tool is written with a single use case in mind which is fullfilled.
It may be extended or not depending on whether I have further use cases.
