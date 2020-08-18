### td-agent-bit.conf

```
[INPUT]
    Name           exec
    Tag            fpga
    Command        nc -U /var/run/fpga0.sock
    Interval_Sec   1
    Interval_NSec  0
    Buf_Size       1kb
    Parser fpga_metric_stat_parser
```
### parsers.conf

```
[PARSER]
    Name        fpga_metric_stat_parser
    Format      regex
    Regex       ^Temperature\:(?<m_temperature>[0-9]*\.[0-9]*) vccint:(?<m_vccint>[0-9]*\.[0-9]*) vccaux:(?<m_vccaux>[0-9]*\.[0-9]*) vccbram:(?<m_vccbram>[0-9]*\.[0-9]*) count:(?<m_count>[0-9]*)$ 
```