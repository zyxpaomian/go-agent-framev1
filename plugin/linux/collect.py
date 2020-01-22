#! /usr/bin/python

import sys
import os

collectitem = sys.argv[1]

class Collectswitch:
    li = {}
    def __init__(self,moden):
        self.li = {
        'hostnamecollect': self.__hostnameCollect,
        'currentusercollect': self.__currentuserCollect,
        }
        function = self.li.get(moden,self.unKnownCollect)
        function()
    def __hostnameCollect(self):
        print(os.popen("hostname").readlines()[0].strip("\n"))

    def __currentuserCollect(self):
        print(os.popen("whoami").readlines()[0].strip("\n"))

    def unKnownCollect(self):
        print("Unkown Collect Item")


Collectswitch(collectitem)
