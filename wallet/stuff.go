package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/calibrae-project/spawn/lib/btc"
	"github.com/calibrae-project/spawn/lib/others/ltc"
	"github.com/calibrae-project/spawn/lib/others/sys"
)

// Cache for txs from already loaded from balance/ folder
var loadedTxs = make(map[[32]byte]*btc.Tx)

// Read a line from stdin
func getline() string {
	li, _, _ := bufio.NewReader(os.Stdin).ReadLine()
	return string(li)
}

func askYesNo(msg string) bool {
	for {
		fmt.Print(msg, " (y/n) : ")
		l := strings.ToLower(getline())
		if l == "y" {
			return true
		} else if l == "n" {
			return false
		}
	}
	// return false
}

func getpass() []byte {
	var pass [1024]byte
	var n int
	var e error
	var f *os.File

	if stdin {
		if *ask4pass {
			fmt.Println("ERROR: Both -p and -stdin switches are not allowed at the same time")
			return nil
		}
		d, er := ioutil.ReadAll(os.Stdin)
		if er != nil {
			fmt.Println("Reading from stdin:", e.Error())
			return nil
		}
		n = len(d)
		if n <= 0 {
			return nil
		}
		copy(pass[:n], d)
		sys.ClearBuffer(d)
		goto check_pass
	}

	if !*ask4pass {
		f, e = os.Open(PassSeedFilename)
		if e == nil {
			n, e = f.Read(pass[:])
			f.Close()
			if n <= 0 {
				return nil
			}
			goto check_pass
		}

		fmt.Println("Seed file", PassSeedFilename, "not found")
	}

	fmt.Print("Enter your wallet's seed password: ")
	n = sys.ReadPassword(pass[:])
	if n <= 0 {
		return nil
	}

	if *list {
		if !*singleask {
			fmt.Print("Re-enter the seed password (to be sure): ")
			var pass2 [1024]byte
			p2len := sys.ReadPassword(pass2[:])
			if p2len != n || !bytes.Equal(pass[:n], pass2[:p2len]) {
				sys.ClearBuffer(pass[:n])
				sys.ClearBuffer(pass2[:p2len])
				println("The two passwords you entered do not match")
				return nil
			}
			sys.ClearBuffer(pass2[:p2len])
		}
		if *list {
			// Maybe he wants to save the password?
			if askYesNo("Save the password on disk, so you won't be asked for it later?") {
				e = ioutil.WriteFile(PassSeedFilename, pass[:n], 0600)
				if e != nil {
					fmt.Println("WARNING: Could not save the password", e.Error())
				} else {
					fmt.Println("The seed password has been stored in", PassSeedFilename)
				}
			}
		}
	}
check_pass:
	for i := 0; i < n; i++ {
		if pass[i] < ' ' || pass[i] > 126 {
			fmt.Println("WARNING: Your secret contains non-printable characters")
			break
		}
	}
	outpass := make([]byte, n+len(secretSeed))
	if len(secretSeed) > 0 {
		copy(outpass, secretSeed)
	}
	copy(outpass[len(secretSeed):], pass[:n])
	sys.ClearBuffer(pass[:n])
	return outpass
}

// return the change addrress or nil if there will be no change
func getChangeAddr() (chng *btc.Addr) {
	if *change != "" {
		var e error
		chng, e = btc.NewAddrFromString(*change)
		if e != nil {
			println("Change address:", e.Error())
			cleanExit(1)
		}
		assertAddressVersion(chng)
		return
	}

	// If change address not specified, send it back to the first input
	for idx := range unspentOuts {
		uo := getUO(&unspentOuts[idx].TxPrevOut)
		if k := pkscrToKey(uo.PkScript); k != nil {
			chng = k.Addr
			return
		}
	}

	fmt.Println("ERROR: Could not determine address where to send change. Add -change switch")
	cleanExit(1)
	return
}

func rawTxFromFile(fn string) *btc.Tx {
	dat := sys.GetRawData(fn)
	if dat == nil {
		fmt.Println("Cannot fetch raw transaction data")
		return nil
	}
	tx, txle := btc.NewTx(dat)
	if tx != nil {
		tx.SetHash(dat)
		if txle != len(dat) {
			fmt.Println("WARNING: Raw transaction length mismatch", txle, len(dat))
		}
	}
	return tx
}

// Get tx with given id from the balance folder, of from cache
func txFromBalance(txid *btc.Uint256, errorIsFatal bool) (tx *btc.Tx) {
	if tx = loadedTxs[txid.Hash]; tx != nil {
		return // we have it in cache already
	}
	fn := "balance/" + txid.String() + ".tx"
	buf, er := ioutil.ReadFile(fn)
	if er == nil && buf != nil {
		var th [32]byte
		btc.ShaHash(buf, th[:])
		if txid.Hash == th {
			tx, _ = btc.NewTx(buf)
			if errorIsFatal && tx == nil {
				println("Transaction is corrupt:", txid.String())
				cleanExit(1)
			}
		} else if errorIsFatal {
			println("Transaction file is corrupt:", txid.String())
			cleanExit(1)
		}
	} else if errorIsFatal {
		println("Error reading transaction file:", fn)
		if er != nil {
			println(er.Error())
		}
		cleanExit(1)
	}
	loadedTxs[txid.Hash] = tx // store it in the cache
	return
}

// Look for specific TxPrevOut in the balance folder
func getUO(pto *btc.TxPrevOut) *btc.TxOut {
	if _, ok := loadedTxs[pto.Hash]; !ok {
		loadedTxs[pto.Hash] = txFromBalance(btc.NewUint256(pto.Hash[:]), true)
	}
	return loadedTxs[pto.Hash].TxOut[pto.Vout]
}

// version byte for P2KH addresses
func verPubkey() byte {
	if litecoin {
		return ltc.AddrVerPubkey(testnet)
	}
	return btc.AddrVerPubkey(testnet)
}

// version byte for P2SH addresses
func verScript() byte {
	// for litecoin the version is identical
	return btc.AddrVerScript(testnet)
}

// version byte for private key addresses
func verSecret() byte {
	return verPubkey() + 0x80
}

// get Addr from PkScript
func addrFromPkscr(scr []byte) *btc.Addr {
	if litecoin {
		return ltc.NewAddrFromPkScript(scr, testnet)
	}
	return btc.NewAddrFromPkScript(scr, testnet)
}

// make sure the version byte in the given address is what we expect
func assertAddressVersion(a *btc.Addr) {
	if a.SegwitProg != nil {
		if a.SegwitProg.HRP != btc.GetSegwitHRP(testnet) {
			println("Sending address", a.String(), "has an incorrect HRP string", a.SegwitProg.HRP)
			cleanExit(1)
		}
	} else if a.Version != verPubkey() && a.Version != verScript() {
		println("Sending address", a.String(), "has an incorrect version", a.Version)
		cleanExit(1)
	}
}
