set terminal pdf color font "Helvetica,20" size 5,4

set datafile separator ","

set style line 1 lc rgb '#dd181f' lt 1 lw 2 pt 7 pi -0.5 ps 0.5
set style line 2 lc rgb '#0060ad' lt 1 lw 2 pt 5 pi -0.5 ps 0.5

set logscale y
set yrange [50:20000]
set xrange [1:180]

set xlabel "# concurrent requests"
set ylabel "99th %tile latency per request"

set key left top

unset grid

plot "no_cache_latencies_99.csv" title "Without Caching" with linespoints ls 1, "with_cache_latencies_99.csv" title "With Caching" with linespoints ls 2;
