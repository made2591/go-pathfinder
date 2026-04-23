# go-pathfinder

Toy CLI that models on-screen TV keyboards as a weighted state graph and finds
the minimum number of remote-control clicks needed to type a string.

The state of the cursor is modelled as `(layer, row, col, capsLock)` so layer
switches (letters → numbers → symbols) and caps-lock toggles are first-class
edges in the graph, not special cases. Pathfinding is plug-and-play behind a
`Pathfinder` interface; `dijkstra` and `astar` ship in the binary.

## Build & run

```sh
go build ./...
./go-pathfinder -layout qwerty -text "Hello!"
```

## Status

v1 — skeleton, QWERTY layout with caps + numbers + symbols layers, Dijkstra,
A* with a Manhattan heuristic. Companion blog post in progress on
`blog.madeddu.xyz`.
