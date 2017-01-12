package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	directory := flag.String("C", "", "change to directory")
	gist_id := flag.String("f", "", "GitHub Gist id")
	compress := flag.Bool("c", false, "compress files")
	extract := flag.Bool("x", false, "extract files")
	list := flag.Bool("t", false, "list files")

	flag.Parse()

	if *extract && *list {
		log.Fatalln("error: flag: -x conflicts with -t")
	}

	if *directory != "" {
		if err := os.Chdir(*directory); err != nil {
			log.Fatalln(err)
		}
	}

	if *extract || *list {
		if *gist_id == "" {
			log.Fatalln("error: flag: -f must we provided with -x or -t")
		}

		content, err := GistGet(gist_id)
		if err != nil {
			log.Fatalln(err)
		}

		reader, err := Uncompress(content)
		if err != nil {
			log.Fatalln(err)
		}

		if *extract {
			if err := ExtractTar(reader); err != nil {
				log.Fatalln(err)
			}
		} else {
			if err := ListTar(reader); err != nil {
				log.Fatalln(err)
			}
		}

	} else if *compress {
		args := flag.Args()
		if len(args) == 0 {
			log.Fatalln("error: flag: nothing to compress")
		}

		content, err := Compress(flag.Args())
		if err != nil {
			log.Fatalln(err)
		}

		gist, err := GistCreate(content)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("ID: %s\n", *gist.ID)
		fmt.Printf("URL: %s\n", *gist.URL)
	} else {
		flag.Usage()
	}
}
