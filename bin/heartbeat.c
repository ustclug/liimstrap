#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <linux/unistd.h>
#include <linux/kernel.h>
#include <sys/sysinfo.h>
#include <curl/curl.h>

long getUptime()
{
    struct sysinfo s_info;
    int error;
    error = sysinfo(&s_info);
    if(error != 0)
    {
        printf("code error = %d\n", error);
    }
    return s_info.uptime;
}

char *getMAC ()
{
    FILE *fin = fopen("/sys/class/net/eth0/address", "r");

    if (!fin)
        return NULL;
    
    char c, *ret = (char *) malloc(13 * sizeof(char));
    int i = 0;
    while ((c = fgetc(fin)) != EOF && i < 12)
        if (c != ':')
            ret[i++] = c;

    ret[i] = '\0';
    fclose(fin);

    return ret;
}

char *getVersion ()
{
    FILE *fin = fopen("/proc/cmdline", "r");

    if (!fin)
        return NULL;

    char cmdline[300];
    fgets(cmdline, 300, fin);

    fclose(fin);

    char *p = strstr(cmdline, "version=");
    if (!p)
    {
        char *ret = (char*) malloc(strlen("devel" + 1));
        strcpy(ret, "devel");
        return ret;
    }
    else
    {
        char *st = p + strlen("version=");
        while (*p && *p != ' ' && *p != '\n')
            p++;
        *p = '\0';
        
        char *ret = (char *) malloc((p - st) * sizeof(char));
        strcpy(ret, st);
        return ret;
    }
}

int main(void)
{
    char *mac = getMAC(),
         *version = getVersion();
    long uptime = getUptime();

    char post_string[80];
    sprintf(post_string, "mac=%.12s&version=%.20s&uptime=%ld",
                mac, version, uptime);
    printf("POST: %s\n", post_string);

    free(mac);
    free(version);

    CURL *curl;
    CURLcode res;
 
    /* In windows, this will init the winsock stuff */ 
    curl_global_init(CURL_GLOBAL_ALL);
 
    /* get a curl handle */ 
    curl = curl_easy_init();
    if(curl) {
      /* First set the URL that is about to receive our POST. This URL can
         just as well be a https:// URL if that is what should receive the
         data. */ 
      curl_easy_setopt(curl, CURLOPT_URL, "http://pxe.ustc.edu.cn:3000/");
      /* Now specify the POST data */ 
      curl_easy_setopt(curl, CURLOPT_POSTFIELDS, post_string);
 
      /* Perform the request, res will get the return code */ 
      res = curl_easy_perform(curl);
      /* Check for errors */ 
      if(res != CURLE_OK)
        fprintf(stderr, "curl_easy_perform() failed: %s\n",
                curl_easy_strerror(res));
 
      /* always cleanup */ 
      curl_easy_cleanup(curl);
    }
    curl_global_cleanup();

    return 0;
}
