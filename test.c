/*************************************************************************
	> File Name: test.c
	> Author: DennisHuang
	> Mail: dshds1993@163.com
	> Created Time: Wed 16 Oct 2019 06:18:38 PM CST
 ************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
int main(){
    int Num;
    for (Num=1; Num<=20000; Num++){
        int bitesNum=0;
        int bitesSum=0;
        int bitesNumArray[10];
        int temp=Num;

        while(temp > 0){
            bitesNumArray[bitesNum]=temp%10;
            bitesNum++;
            temp /= 10;
        }

        for(int index=0;index<bitesNum;index++){
            //pow3bitesNum=pow(bitesNumArray[index],bitesNum);
            int pow3bitesNum = 1, a;
            for(a=1; a <= bitesNum; a++)
                pow3bitesNum *= bitesNumArray[index];
            bitesSum+=pow3bitesNum;
        }

        if(Num==bitesSum)
            printf("%d\n",Num);
    }
}
