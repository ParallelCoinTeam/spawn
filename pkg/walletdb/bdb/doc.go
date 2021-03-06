/*Package bdb implements an instance of walletdb that uses boltdb for the backing
datastore.

Usage

This package is only a driver to the walletdb package and provides the database type of "bdb". The only parameter the
Open and Create functions take is the database path as a string:

	db, e := walletdb.Open("bdb", "path/to/database.db")
	if e != nil  {
		// Handle error
	}
	db, e := walletdb.Create("bdb", "path/to/database.db")
	if e != nil  {
		// Handle error
	}
*/
package bdb
