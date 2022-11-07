
# YaD is a Yandex.Disk CLI tool v.1.0.0

## Install

```sh
   go get -u github.com/ilyabrin/yad
```

### Generate the initial config file

```sh
   # Create .yad.yaml in current directory
   yad init .

   # or init at home dir
   yad init

   # Note: root rights may be required
   
```

Command above creates the config file

### Paste your credentials into config

TODO: add instruction section
TODO: use .yad.yaml instead

```sh
   ~~yad account add <new_email_here>~~
```

### You can switch between different accounts in config with following command

```sh
   yad switch <account name in config>
```

### Start using yad

```sh
   yad info [ base (default) | all | me | system ]
   yad mkdir sample_folder
   yad meta set PATH_TO_RESOURCE KEY VALUE
   yad meta [ get RESOURCE_PATH ]
   yad upload <http(s) URL here | ./local/path>
   yad download path_to_disk_resource
   yad public [ list | meta | save ]
   yad publish sample_folder
   yad unpublish sample_folder
   yad delete sample_folder
   yad trash [ list (default) | clean | delete ~ remove ]
   yad help
```

## More command

```sh
   yad help
```

## Get CLI tool version by typing

```sh
   yad version
```
