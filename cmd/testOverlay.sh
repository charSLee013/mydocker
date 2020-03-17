#!/usr/bin/env bash
# ----------------------------
# 测试overlay2驱动
# ----------------------------
# 全局设置
OVERLAYDIR="/var/lib/gocker/overlay"
IMAGESHA256="123456"
LOWERDIR1="$OVERLAYDIR/$IMAGESHA256/lowerdir1"
LOWERDIR2="$OVERLAYDIR/$IMAGESHA256/lowerdir2"
LOWERDIR3="$OVERLAYDIR/$IMAGESHA256/lowerdir3"
UPPERDIR="$OVERLAYDIR/$IMAGESHA256/upperdir"
WORKDIR="$OVERLAYDIR/$IMAGESHA256/workdir"
MERGEDDIR="$OVERLAYDIR/$IMAGESHA256/mergeddir"
# ----------------------------

## 创建overlay 文件夹
function createOverlay {
  echo -n "Create overlay in $OVERLAYDIR/$IMAGESHA256..."

  if ! sudo mkdir -p $LOWERDIR1 $LOWERDIR2 $LOWERDIR3 $UPPERDIR $WORKDIR $MERGEDDIR
  then
    echo "[failed]"
    exit
  else
    echo "[sucessed]"
  fi

  
}

## 写入测试文件
function writeTestFile {
  echo -n "Write test file in lowerdir ..."


  if ! echo "I am in lowerdir1/a.txt" | sudo tee $LOWERDIR1/a.txt > /dev/null
  then
    echo "[failed]"
    exit
  fi

  if ! echo "I am in lowerdir2/b.txt" | sudo tee $LOWERDIR2/b.txt > /dev/null
   then
    echo "[failed]"
    exit
  fi


  if ! echo "I am in lowerdir3/a.txt" | sudo tee $LOWERDIR3/a.txt > /dev/null
  then
    echo "[failed]"
    exit
  fi

  echo "[sucessed]"
  
}

## 挂载为overlay filesystem
function mountOverlay {
  echo -n "mount overlay filesystem ..."

## 注意这里的lowerdir层序,从左到右层次越低
sudo mount -t overlay -o\
lowerdir=$LOWERDIR1:$LOWERDIR2:$LOWERDIR3,\
upperdir=$UPPERDIR,\
workdir=$WORKDIR \
none $MERGEDDIR

  if [ $? != 0 ]
  then
    echo "[failed]"
    exit
  fi

  echo "[sucessed]"
  
}

## 查看$MERGEDDIR文件夹内的文件的差别
function checkDiff {
  echo -e "\n\r"
  ## 开始测试
sudo cat <<EOF
 ### #####   #    #####  #####   ##### #####  ### #####
#  #   #     ##    #  #    #       #    #  # #  #   #
###    #    # #    #  #    #       #    ###  ###    #
  ##   #    ####   ###     #       #    #      ##   #
#  #   #   #   #   # ##    #       #    #  # #  #   #
###   ###  #   ## ### ##  ###     ###  ##### ###   ###
EOF


  ## 如果层序没错，那么这里的lowerdir1会被lowder3给覆盖
  echo -n "Diff $LOWERDIR1/a.txt $MERGEDDIR/a.txt : "

  if ! sudo diff $LOWERDIR1/a.txt $MERGEDDIR/a.txt > /dev/null;then
    echo "[ NOT THE SAME ]"
  else
    echo "[ THE SAME ]"
  fi

  ## 检查 merger/a.txt 与 lowder3/a.txt 是否相同
  echo -n "Diff $LOWERDIR3/a.txt $MERGEDDIR/a.txt : "
  
  if ! sudo diff $LOWERDIR3/a.txt $MERGEDDIR/a.txt > /dev/null ;then
    echo "[ NOT THE SAME ]"
  else
    echo "[ THE SAME ]"
  fi

  ## 更改 merger/a.txt 文件后查看与lowerdir3/a.txt 是否一致
  echo "Now, let's modify $MERGEDDIR/b.txt"
  echo "Now,I am in mergeddir" | sudo tee -a $MERGEDDIR/b.txt > /dev/null
  echo -n "Diff $LOWERDIR2/b.txt $MERGEDDIR/b.txt : "

  if ! sudo diff $LOWERDIR2/b.txt $MERGEDDIR/b.txt > /dev/null;then
    echo "[ NOT THE SAME ]"
  else
    echo "[ THE SAME ]"
  fi
}

# 清理现场
function clean {
  sudo umount $MERGEDDIR > /dev/null
  sudo rm -r "$OVERLAYDIR/$IMAGESHA256" > /dev/null
}

# 主函数
function main {
  clean

  # 1. 创建相对应的文件夹
  createOverlay

  # 2. 创建测试文件
  writeTestFile

  # 3. 挂载 overlay filesystem
  mountOverlay

  # 4. 测试 overlay
  checkDiff

  # 5. clean
  clean

  exit
}

main
