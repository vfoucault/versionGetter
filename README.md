# versionGetter


versionGetter walks a terraform directory tree looking for .tf files in order to look for modules and fetch their version.


# Build

go build

# args

```bash
Usage of ./versionGetter:
  -extra-skip value
        Extra skip dirs
  -output string
        output format (text|json) (default "text")
  -source-path string
        The path to the source directory to scan
  -stats
        Show stats
  -verbose
        be verbose
```

`-extra-skip` extra set of directory to skip. By default, all `.git` and `.terraform` are skipped.

`-output` the output format. Either text (default) or json

`-source-path` the source input path to start walk from

`-stats` display stats after processing

`-verbose` be verbose

# Example call

```bash 
versionGetter --source-path ~/projects/kafka
versionGetter 2018/10/15 11:18:20 [INFO] ▶ search in /Users/vianneyfoucault/projects/kafka
+------------------------------------------------------------------------+----------------+----------+------------------------------------------+---------+
|                                       FILE                             |     MODULE     |   TYPE   |                  SOURCE                  | VERSION |
+---------------------------------------------------------------------- -+----------------+----------+------------------------------------------+---------+
| /Users/vianneyfoucault/projects/kafka/kafka-front/terraform/main.tf    | network_finder | ssh      | remote_modules.git                       | 1.0.9   |
| /Users/vianneyfoucault/projects/kafka/kafka-front/terraform/main.tf    | security_group | registry | terraform-aws-modules/security-group/aws |    1.20 |
| /Users/vianneyfoucault/projects/kafka/kafka-front/terraform/main.tf    | iam_role       | ssh      | remote_modules.git                       | 1.0.9   |
| /Users/vianneyfoucault/projects/kafka/kafka-front/terraform/main.tf    | kafka          | ssh      | remote_modules.git                       | 1.0.9   |
| /Users/vianneyfoucault/projects/kafka/kafka-front/terraform/main.tf    | data_volume    | ssh      | remote_modules.git                       | 1.0.9   |
+------------------------------------------------------------------------+----------------+----------+------------------------------------------+---------+

```

