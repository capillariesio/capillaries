#!/usr/bin/python

import sys
import csv

csv.field_size_limit(1024*1024*10)

reader = csv.reader( sys.stdin,delimiter=',', escapechar='\\', quotechar='"')
row_idx = 0
for row in reader:
    if len(row) == 0:
        continue
    if row_idx == 0:
        print( '| ' + ' | '.join(row) + " |" )
        print( '| ' + ' | '.join(['---' for field in row]) + " |" )
    elif row_idx < 11:
        vals = []
        for val in row:
            if len(val) > 100:
                vals.append(val[:100] + " ... total length " + str(len(val)))
            else:
                vals.append(val)
        print( "| " + " | ".join(vals) + " |" )
    row_idx += 1
print( f"\nTotal {row_idx - 1} rows" )
