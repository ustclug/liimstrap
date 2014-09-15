#include <stdio.h>
#include <stdlib.h>
#include <X11/Xlib.h>
#include <X11/extensions/scrnsaver.h>

/* Report amount of X server idle time. */
/* Build with- */
/* cc xidle.c -o xidle -lX11 -lXext -lXss */

int main(int argc, char *argv[])
{
    Display *display;
    int event_base, error_base;
    XScreenSaverInfo info;
    unsigned int ms;

    if (argc != 3)
    {
        fprintf(stderr, "Usage: %s [DISPLAY] [Xauthority]\n", argv[0]);
        return 1;
    }

    setenv("DISPLAY", argv[1], 1);
    setenv("XAUTHORITY", argv[2], 2);
    display = XOpenDisplay("");

    if(display)
    {
        if (XScreenSaverQueryExtension(display, &event_base, &error_base))
        {
            XScreenSaverQueryInfo(display, DefaultRootWindow(display), &info);

            ms = (unsigned int)info.idle;
            printf("%u\n", ms);
            return(0);
        }
        else
        {
            fprintf(stderr,"Error: XScreenSaver Extension not present\n");
            return(1);

        }
    }
    else
    {
        fprintf(stderr,"Error: Invalid Display\n");
        return(1);
    }

    return 0;
}
