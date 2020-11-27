
# YaD is a Yandex.Disk CLI tool v.1

## Install

```sh
   go get -u github.com/ilyabrin/yad
```

### Generate the initial config file:

```sh
   yad init .
```
Command above creates the config file

### Paste your credentials into config:

```sh
   yad account add <new_email_here>
```

### You can switch between different accounts in config with following command:

```sh
   yad switch <account name in config>
```

### Start using yad:

```sh
   yad mkdir sample_folder
   yad meta set KEY VALUE
   yad meta get KEY
   yad upload __<paste fule URL here>__
   yad publish sample_folder
   yad unpublish sample_folder
   yad delete sample_folder
```

## More command
```sh
   yad help
```

## Get CLI tool version by typing:
```sh
   yad version
```
