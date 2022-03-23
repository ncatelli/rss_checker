#!/bin/sh

newitems=$(./rss_checker)
echo ::set-output name=new_items::$newitems
