package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type homelabProjectVars struct {
	Tailscale struct {
		TailnetName    string
		ServerOAuthKey struct {
			Secret string
		}
		K8SOperatorOAuthKey struct {
			ClientID string
			Secret   string
		}
	}
	K3S struct {
		Server struct {
			Name string
		}
		JoinToken string
	}
	SMBServer struct {
		Username string
		Password string
	}
	Netdata struct {
		Parent struct {
			Token string
			Rooms string
		}
		Child struct {
			Token string
			Rooms string
		}
	}
	Forgejo struct {
		SecretKey     string
		InternalToken string
	}
}

func main() {
	var homelab = &homelabProjectVars{}
	homelab.SMBServer.Username = os.Getenv("SMB_SERVER_USERNAME")
	homelab.SMBServer.Password = os.Getenv("SMB_SERVER_PASSWORD")
	homelab.Tailscale.TailnetName = os.Getenv("TS_TAILNET")
	homelab.Tailscale.ServerOAuthKey.Secret = os.Getenv("TS_SECRET_TAG_SERVER")
	homelab.Tailscale.K8SOperatorOAuthKey.ClientID = os.Getenv("TS_CLIENTID_K8S_OPERATOR")
	homelab.Tailscale.K8SOperatorOAuthKey.Secret = os.Getenv("TS_SECRET_K8S_OPERATOR")
	homelab.Netdata.Parent.Token = os.Getenv("NETDATA_TOKEN")
	homelab.Netdata.Parent.Rooms = os.Getenv("NETDATA_ROOMS")
	homelab.Netdata.Child.Token = os.Getenv("NETDATA_TOKEN")
	homelab.Netdata.Child.Rooms = os.Getenv("NETDATA_ROOMS")
	homelab.K3S.Server.Name = "k3s-master"
	homelab.Forgejo.SecretKey = os.Getenv("FORGEJO_SECRET_KEY")
	homelab.Forgejo.InternalToken = os.Getenv("FORGEJO_INTERNAL_TOKEN")

	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if !isTemplateFile(d) {
			return nil
		}
		log.Printf("found template file %s in %s", d.Name(), filepath.Dir(path))
		err = addToGitIgnore(path)
		if err != nil {
			log.Fatalf("cannot add to file %s to gitignore: %s", d.Name(), err)
		}
		err = applyTemplate(path, homelab)
		if err != nil {
			log.Fatalf("cannot apply template of file %s: %s", d.Name(), err)
		}
		return nil
	})
}

func isTemplateFile(d fs.DirEntry) bool {
	return !d.IsDir() && filepath.Ext(d.Name()) == ".tmpl"
}

func addToGitIgnore(tmplPath string) error {
	dir := filepath.Dir(tmplPath)
	fileName := strings.TrimSuffix(filepath.Base(tmplPath), filepath.Ext(tmplPath))
	gitIgnorePath := fmt.Sprintf("%s/.gitignore", dir)

	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		log.Printf("creating file %s", gitIgnorePath)
		file, err := os.Create(gitIgnorePath)
		if err != nil {
			return err
		}
		defer file.Close()
		fmt.Fprintln(file, "# This file has been generated, do not edit.")
		fmt.Fprintf(file, "%s\n", fileName)
	} else {
		fileBytes, err := os.ReadFile(gitIgnorePath)
		if err != nil {
			return err
		}
		if strings.Contains(string(fileBytes), fileName) {
			log.Printf("file %s already in %s", fileName, gitIgnorePath)
			return nil
		}
		file, err := os.OpenFile(gitIgnorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = fmt.Fprintf(file, "%s\n", fileName)
		if err != nil {
			return err
		}
	}

	log.Printf("file %s added to %s sucessfully", fileName, gitIgnorePath)
	return nil
}

func applyTemplate(tmplPath string, tmplVars *homelabProjectVars) error {
	dir := filepath.Dir(tmplPath)
	fileName := strings.TrimSuffix(filepath.Base(tmplPath), filepath.Ext(tmplPath))

	tmplBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(dir, fileName))
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New(filepath.Join(dir, fileName)).Parse(string(tmplBytes))
	if err != nil {
		return err
	}

	return tmpl.Execute(file, tmplVars)
}
