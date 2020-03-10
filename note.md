### NOTE

---------------------

#### mount overlay2

##### Create a overlay filesystem

```bash
#!/bin/bash
## need kernel 4.0+

## global set
overlayDir="/var/lib/gocker/overlay"
imageSha256="123456"

lowerdir="$overlayDir/$imageSha256/lowerdir"
upperdir="$overlayDir/$imageSha256/upperdir"
workdir="$overlayDir/$imageSha256/workdir"
mergeddir="$overlayDir/$imageSha256/mergeddir"

## clean
umount $mergeddir > /dev/null 2>&1

## create floder
mkdir -p $lowerdir $upperdir $workdir $mergeddir

## mount
sudo mount -t overlay -o \
lowerdir=$lowerdir,\
upperdir=$upperdir,\
workdir=$workdir \
none $mergeddir

```

##### test file write

```bash
#!/bin/bash

## global set
overlayDir="/var/lib/gocker/overlay"
imageSha256="123456"

lowerdir="$overlayDir/$imageSha256/lowerdir"
upperdir="$overlayDir/$imageSha256/upperdir"
workdir="$overlayDir/$imageSha256/workdir"
mergeddir="$overlayDir/$imageSha256/mergeddir"

## clean
find $overlayDir -type f | xargs rm -f > /dev/null 2>&1
bash ./removeOverlay2.sh

## create
bash ./mountOverlay2.sh

## write a file to lowerdir
echo 'Hello World' > $lowerdir/a.txt

## check consistency
diff $lowerdir/a.txt $mergeddir/a.txt
 if [ $? != 0 ]

then

    echo "Different!"

else

    echo "Same!"

fi

## change file from upperdir
echo 'Hi' >> $mergeddir/a.txt

## check again
diff $lowerdir/a.txt $mergeddir/a.txt
```

##### remove overlay filesystem

```bash
#!/bin/bash

overlayDir="/var/lib/gocker/overlay"
imageSha256="123456"

lowerdir="$overlayDir/$imageSha256/lowerdir"
upperdir="$overlayDir/$imageSha256/upperdir"
workdir="$overlayDir/$imageSha256/workdir"
mergeddir="$overlayDir/$imageSha256/mergeddir"


## umount
umount $mergeddir > /dev/null 2>&1

## delete folder
rm -rf "$overlayDir/$imageSha256"
```