# Googol

``Googol`` is a well-simple program that outputs generations of Conway's Game of Life as GIF animation. The name is a
lousy acronym which stands for ``Goo``gol is yet another ``g``ame ``o``f ``l``ife. It was written in Golang, thus some
gophers certainly will prefer ``G``ame ``o``f Life... err..  ``o``h! in ``gol``ang...

If you are hooked on Game of Life maybe you should like also check another implementation of mine of this game by using
``IA-32`` assembly [here](https://github.com/rafael-santiago/life).

## How can I build it?

You need Golang installed. You can get it installed by [accessing](https://golang.org/dl).

Being a Golang program, you can only use:

```
    you@somewhere:~/over/the/rainbow# go run googol.go [sub-command] [options]
    you@somewhere:~/over/the/rainbow# _
```

If you actually want to its binary, use:

```
    you@somewhere:~/over/the/rainbow# go build googol.go
    you@somewhere:~/over/the/rainbow# ./googol [sub-command] [options]
    you@somewhere:~/over/the/rainbow# _
```

If you want to install it, you will need to install [Hefesto](https://github.com/rafael-santiago/hefesto).

Once Hefesto well-installed and running. Do the following:

```
    you@somewhere:~/over/the/rainbow# mkdir temp-mess
    you@somewhere:~/over/the/rainbow/temp-mess# git clone https://github.com/rafael-santiago/helios
    you@somewhere:~/over/the/rainbow/temp-mess/helios# hefesto --install=go-toolset
    you@somewhere:~/over/the/rainbow/temp-mess/helios# cd ../..
    you@somewhere:~/over/the/rainbow# rm -rf temp-mess
```

Now your Hefesto copy knows how to build a Golang program. If you want to build the program by using Hefesto:

```
    you@somewhere:~/over/the/rainbow# hefesto
    you@somewhere:~/over/the/rainbow# _
```

In order to install Googol on your system:

```
    you@somewhere:~/over/the/rainbow# hefesto --install
    you@somewhere:~/over/the/rainbow# _
```

Uninstalling:

```
    you@somewhere:~/over/the/rainbow# hefesto --uninstall
    you@somewhere:~/over/the/rainbow# _
```

## How can I play with it?

If you do not know what Game of Life is take a look at [here](https://www.conwaylife.com).

Until now googol can be used in two ways. As a batch tool that can creates GIF images representing the game's generations
or as a webserver with a web page where you can input game data and see its output.

### Playing with it in batch mode

Use the sub-command ``gif``:

```
    you@somewhere:~/over/the/rainbow# googol gif > big-blank.gif
    you@somewhere:~/over/the/rainbow# _
```

This is the shortest way of using the gif command. However, it will produce game generations that are nothing else that a
big boredom. You need to define the initial game board state. The way of doing it is by passing a set of options in the
form ``--<n>,<n>.``, where <n> denotes the alive cell coordinates.

Let's suppose we want to define a blinker:

```
    you@somewhere:~/over/the/rainbow# googol gif --2,2. --2,3. --2,4. > blinker.gif
    you@somewhere:~/over/the/rainbow# _
```

The default output is stdout, but you can change it by passing ``--out=<file-path>``:

```
    you@somewhere:~/over/the/rainbow# googol gif --2,2. --2,3. --2,4. \
    > --out=/usr/share/docs/gifs/conways-game-of-life-blinker.gif
    you@somewhere:~/over/the/rainbow# _
```

The GIF animation is not endless by default. In order to make it endless pass the option ``--endless``.

```
    you@somewhere:~/over/the/rainbow# googol gif --2,2. --2,3. --2,4. \
    > --endless > blinker.gif
    you@somewhere:~/over/the/rainbow# _
```

This is the basic usage, anyway there are a bunch of other options accepted by ``gif`` sub-command. If you want to learn
more get the gif's command guide in the following way:

```
    you@somewhere:~/over/the/rainbow# googol help gif
    you@somewhere:~/over/the/rainbow# _
```

The following command generates the ``acorn.gif`` pattern in Figure 1.

```
    you@somewhere:~/over/the/rainbow# googol gif \
    > --50,50. --51,52. --52,49. --52,50. --52,53. --52,54. --52,55. \
    > --board-width=100 --board-height=100 \
    > --cell-size-inpx=2 --delay=1 --gen-total=5206 \
    > --gif-width=200 --gif-height=200 --endless
    you@somewhere:~/over/the/rainbow# _
```

**Figure 1**: Acorn pattern.

![Acorn](https://github.com/rafael-santiago/googol/blob/master/etc/acorn.gif)

### Playing with it in httpd mode

Use the sub-command ``httpd``:

```
    you@somewhere:~/over/the/rainbow# googol httpd
    you@somewhere:~/over/the/rainbow# _
```

If you run the command above, a webserver at ``http://localhost:8080/googol`` will be created. Only yoy will be able to
access it. If you want to other people accessing it you must to pass the interface address:

```
    you@somewhere:~/over/the/rainbow# googol httpd --addr=<your-ip-or-hostname>
    you@somewhere:~/over/the/rainbow# _
```

In order to set another port than 8080:

```
    you@somewhere:~/over/the/rainbow# googol httpd --addr=<your-ip-or-hostname> \
    > --port=101
    you@somewhere:~/over/the/rainbow# _
```

It is also possible to have a secure httpd by passing ``--https`` option flag, but in this case will be necessary to pass
``--server-crt`` and ``--server-key`` options, too. The ``--server-crt`` must point to a valid crt file and the
``--server-key`` must point to a valid private key file.

```
    you@somewhere:~/over/the/rainbow# googol httpd --addr=<your-ip-or-hostname> \
    > --port=101 --server-crt=etc/googol.crt --server-key=etc/googol.key --https
    you@somewhere:~/over/the/rainbow# _
```

In order to define a default alive cells set pass options in the same form that gif command expects (``--<n>,<n>.``).

If you want to change the (lousy) HTML form template, use the option ``--form-template``:

    you@somewhere:~/over/the/rainbow# googol httpd \
    > --form-template=etc/poetry-in-html/your-utter-awesome-googol-template.html
    you@somewhere:~/over/the/rainbow# _

Table 1 lists all available template actions.

**Table 1**: All available HTML template actions.

| Action | Expands to |
|:------:|:-----------|
|``{{.Proto}}``|``http`` or ``https`` depending on ``--https`` option flag|
|``{{.Addr}}``|the server address|
|``{{.Port}}``| the server port|
|``{{.InitialState}}``|lists the set of initial alive cells (iterate over by using .range)|
|``{{.BoardWidth}}``|the board width|
|``{{.BoardHeight}}``|the board height|
|``{{.GIFWidth}}``|the GIF width|
|``{{.GIFHeight}}``|the GIF height|
|``{{.Delay}}``|the animation delay|
|``{{.CellSizeInPx}}``|the number of pixels per board cell|
|``{{.GenTotal}}``|the total of game generations|
|``{{.BkColor}}``|a HTML select field which lists all available background colors|
|``{{.FgColor}}``|a HTML select field which lists all available foreground colors|
|``{{.Endless}}``|the current state of '--endless' flag (for the current game instance)|
|``{{.Error}}``|an error message when occurred one|
|``{{.GIFData}}``|GIF image encoded in radix/base-64|

The best way of understanding how to deal with those template actions is by reading ``etc/template.html``.

The sub-command ``httpd`` also accepts a bunch of other commands that you can learn more by running its command guide:

```
    you@somewhere:~/over/the/rainbow# googol help httpd
    you@somewhere:~/over/the/rainbow# _
```

In Figure 2 you can see the ``Googol``'s default HTML interface.

**Figure 2**: Default HTML interface.

![HTTPd-default-interface](https://github.com/rafael-santiago/googol/blob/master/etc/httpd-screenshot.png)
