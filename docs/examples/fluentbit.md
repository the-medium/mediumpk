### td-agent-bit.conf

```
[SERVICE]
    parsers_file  parsers.conf

[INPUT]
    Name          exec
    Tag           mbpu.[MBPU_INDEX].${HOSTNAME}
    Command       nc -U /var/run/mbpu[MBPU_INDEX].sock
    Interval_Sec  1
    Interval_NSec 0
    Buf_Size      1kb
    Parser        json

[OUTPUT]
    Name    stdout
    Match    *

[OUTPUT]
    Name    forward
    Match    *
    Host    FLUENTD_IP_ADDRESS
    Port    24224
```
