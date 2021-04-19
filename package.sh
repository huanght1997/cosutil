#!/bin/sh -x
if ! go build -ldflags "-s -w" -o ./cosutil; then
  echo "make error"
  exit 1
fi

cosutil_version=$(./cosutil -v | cut -d' ' -f3)
echo "build version: $cosutil_version"
rm cosutil

rm -rf ./release
mkdir -p ./release

make

os_all='linux windows freebsd'
arch_all='386 amd64 arm arm64'

cd ./release || exit 1

for os in $os_all; do
  for arch in $arch_all; do
    path="cosutil_${cosutil_version}_${os}_${arch}"

    if [ "${os}" = "windows" ]; then
      if [ ! -f "./cosutil_${os}_${arch}.exe" ]; then
        continue
      fi
      mkdir "${path}"
      mv "./cosutil_${os}_${arch}.exe" "${path}/cosutil.exe"
    else
      if [ ! -f "./cosutil_${os}_${arch}" ]; then
        continue
      fi
      mkdir "${path}"
      mv "./cosutil_${os}_${arch}" "${path}/cosutil"
    fi
    cp ../LICENSE "${path}"
    cp ../README.md "${path}"

    if [ "${os}" = "windows" ]; then
      zip -rq "${path}.zip" "${path}"
    else
      tar -zcf "${path}.tar.gz" "${path}"
    fi
    rm -rf "${path}"
  done
done

cd - || exit 1
