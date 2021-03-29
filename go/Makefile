NAME=gosb

all: $(NAME)

$(NAME):
	sh compile.sh
	sh clean.sh
	sh install.sh

.PHONY: clean

clean:
	@sh clean.sh
	@rm -f gosb_run.sh
