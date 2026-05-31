// 算法笔记基础知识 实现简单的冒泡排序
#include <stdio.h>
#include <stdlib.h>

int main () {
    int a[10] = {3,1 ,4, 5, 2};

    for (int i = 1; i <= 4; ++i) {
        for (int j = 0; j < 5 - i; ++j) {
            if (a[j] > a[j+1]) {
                int temp = a[j];
                a[j] = a[j+1];
                a[j+1] = temp;
            }
        }
    }

    // 打印最终的结果
    for (int i = 0; i < 5; ++i) {
        printf("%d ", a[i]);
    }
    printf("\n");


    int b[5][6] = {{3,1,2}, {8, 4}, {}, {1,2,3,4,5}};
    for (int i = 0; i < 5; ++i) {
        for (int j = 0; j < 6; ++j) {
            printf("%4d ", b[i][j]);
        }
        printf("\n");
    }
    printf("\n");

    return 0;
}


