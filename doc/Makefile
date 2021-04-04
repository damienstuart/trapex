# Makefile for trapex
#

# If go is not in your path, you can put the full pathname here
A2X = $(shell which a2x)

MAN_SRC = trapex.man.asciidoc
MANPAGE = trapex.8
MANPDF = trapex.man.pdf

all: pdf

man: $(MAN_SRC)
	$(A2X) -L -f manpage $(MAN_SRC) 

pdf: man
	(groff -man $(MANPAGE) | ps2pdf - $(MANPDF)) || true

clean:
	/bin/rm -f $(MANPAGE) $(MANPDF)


###EOF###