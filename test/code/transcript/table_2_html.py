#!/usr/bin/python

import sys
import csv

csv.field_size_limit(1024*1024*10)

print('<table class="table table-striped text-right" style="font-size:smaller">')

reader = csv.reader( sys.stdin,delimiter=',', escapechar='\\', quotechar='"')
row_idx = 0
for row in reader:
    if len(row) == 0:
        continue
    if row_idx == 0:
        print( '<thead><th class="text-right">' + '</th><th class="text-right">'.join(row) + "</th></thead><tbody>" )
    elif row_idx < 11:
        vals = []
        for val in row:
            if len(val) > 100:
                vals.append(val[:100] + " ... total length " + str(len(val)))
            else:
                vals.append(val)
        print( "<tr><td>" + "</td><td>".join(vals) + "</td></tr>" )
    row_idx += 1
print( "</tbody></table>")
print( f"<p>Total {row_idx - 1} rows</p>" )
