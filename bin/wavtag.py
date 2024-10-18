import os
# https://pypi.org/project/pytaglib/
import taglib


def resave_metadata(file_path):
    song = taglib.File(file_path)
    print('Tags=',song.tags)
    if "TITLE" in song.tags:
        song.tags["TITLE"] = song.tags["TITLE"]
    if "ALBUM" in song.tags:
        song.tags["ALBUM"] = song.tags["ALBUM"]
#         a=song.tags["ALBUM"][0].split("&")
#         song.tags["ALBUM"][0]=a[0]
    if "ALBUMARTIST" in song.tags:
        song.tags["ALBUMARTIST"] = song.tags["ALBUMARTIST"]
    if "ALBUMARTISTSORT" in song.tags:
        song.tags["ALBUMARTISTSORT"] = song.tags["ALBUMARTISTSORT"]
    if "ARTIST" in song.tags:
        song.tags["ARTIST"] = song.tags["ARTIST"]
#         a=song.tags["ARTIST"][0].split("&")
#         song.tags["ARTIST"][0]=a[0]
    if "ARTISTS" in song.tags:
        song.tags["ARTISTS"] = song.tags["ARTISTS"]
    if "ARTISTSORT" in song.tags:
        song.tags["ARTISTSORT"] = song.tags["ARTISTSORT"]
    if "LABEL" in song.tags:
        song.tags["LABEL"] = song.tags["LABEL"]

#     song.tags["COMMENT"][0] = ''
#     print("after tags=",song.tags)
    song.save()
        

dir = r"/mnt/e/test"

all_sub_dir = []

dir_contests = os.walk(dir)
for root, dirs, files in dir_contests:
    print(root)
    print(dirs)
    print(files)
    for dirname in dirs:
        dir_path = os.path.join(root, dirname)
        print(dir_path)
        for f in os.listdir(dir_path):
            if not f.endswith(".wav"):   # 这里限定仅处理 wav 格式音乐
                continue
            file_path = os.path.join(dir_path, f)
            print(file_path)

            resave_metadata(file_path)