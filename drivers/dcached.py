#encoding: utf-8

from posixpath import join as urljoin
import requests

class DCached(object):

	def __init__(self, host='http://127.0.0.1'):

		self.host = host


	def __getattr__(self, method):

		def fun(*args, **kwargs):

			url = urljoin(self.host, 'cache', method, '/'.join(args))
			print url

			response = None
			try:
				response = requests.post(url, json=kwargs)
				r_json = response.json()
			except Exception as e:
				if response != None:
					return response.status_code, response.text
				else:
					raise

			return response.status_code, r_json

		return fun




if __name__ == '__main__':
	import time

	cache = DCached('http://10.0.3.19:8090')


	status, response = cache.remove('key', appname='PythonDriver', key='name')
	print(status, response)

	status, response = cache.set(appname='PythonDriver', key='name', value='diego', ttl=10)
	print(status, response)

	status, response = cache.get(appname='PythonDriver', key='name')
	print(status, response)

	time.sleep(10)
	status, response = cache.get(appname='PythonDriver', key='name')
	print(status, response)

	status, response = cache.pepe(appname='PythonDriver', key='name')
	print(status, response)



