# ALEXATOP
I put this tool together over a weekend in order to produce a dataset for other tools I'm writing. We PoC this against the Alexa top 1m, but you might enjoy using it to find domains that land in other ranges of interest.

Use it like so;
```sh
wget http://s3.amazonaws.com/alexa-static/top-1m.csv.zip
unzip top-1m.csv.zip
cut -d, -f2- top-1m.csv > top1m.txt
./ALEXATOP -nameFile top1m.txt -n 50 -threads 1000 2>/dev/null
```

Check out the corresponding [post.](https://dualuse.io/blog/alexatop/)

# REQUIREMENTS

None.

# BUILDING

```sh
go build
```

# COMPLAINTS

Yes.

# LICENSE

MIT.