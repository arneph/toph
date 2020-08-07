// Test file to check expected behaviour

#include <stdio.h>
#include <unistd.h>

int main() {
	for (;;) {
		printf("tick: %ld\n", sysconf(_SC_CLK_TCK));
	}
	return 0;
}
