#!/usr/bin/env bash
# jqschema.sh
jq -c '
def walk(f):
  . as $in |
  if type == "object" then
    reduce keys[] as $k ({}; . + {($k): ($in[$k] | walk(f))})
  elif type == "array" then
    if length == 0 then [] else [.[0] | walk(f)] end
  else
    type
  end;
walk(.)
'
