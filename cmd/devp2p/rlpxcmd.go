// Copyright 2020 The go-bitone Authors
// This file is part of go-bitone.
//
// go-bitone is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-bitone is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-bitone. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"net"
	"os"

	"github.com/bitone/go-bitone/cmd/devp2p/internal/ethtest"
	"github.com/bitone/go-bitone/crypto"
	"github.com/bitone/go-bitone/internal/utesting"
	"github.com/bitone/go-bitone/p2p"
	"github.com/bitone/go-bitone/p2p/rlpx"
	"github.com/bitone/go-bitone/rlp"
	"gopkg.in/urfave/cli.v1"
)

var (
	rlpxCommand = cli.Command{
		Name:  "rlpx",
		Usage: "RLPx Commands",
		Subcommands: []cli.Command{
			rlpxPingCommand,
			rlpxEthTestCommand,
		},
	}
	rlpxPingCommand = cli.Command{
		Name:   "ping",
		Usage:  "ping <node>",
		Action: rlpxPing,
	}
	rlpxEthTestCommand = cli.Command{
		Name:      "eth-test",
		Usage:     "Runs tests against a node",
		ArgsUsage: "<node> <path_to_chain.rlp_file>",
		Action:    rlpxEthTest,
		Flags:     []cli.Flag{testPatternFlag},
	}
)

func rlpxPing(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", n.IP(), n.TCP()))
	if err != nil {
		return err
	}
	conn := rlpx.NewConn(fd, n.Pubkey())
	ourKey, _ := crypto.GenerateKey()
	_, err = conn.Handshake(ourKey)
	if err != nil {
		return err
	}
	code, data, _, err := conn.Read()
	if err != nil {
		return err
	}
	switch code {
	case 0:
		var h ethtest.Hello
		if err := rlp.DecodeBytes(data, &h); err != nil {
			return fmt.Errorf("invalid handshake: %v", err)
		}
		fmt.Printf("%+v\n", h)
	case 1:
		var msg []p2p.DiscReason
		if rlp.DecodeBytes(data, &msg); len(msg) == 0 {
			return fmt.Errorf("invalid disconnect message")
		}
		return fmt.Errorf("received disconnect message: %v", msg[0])
	default:
		return fmt.Errorf("invalid message code %d, expected handshake (code zero)", code)
	}
	return nil
}

func rlpxEthTest(ctx *cli.Context) error {
	if ctx.NArg() < 3 {
		exit("missing path to chain.rlp as command-line argument")
	}

	suite := ethtest.NewSuite(getNodeArg(ctx), ctx.Args()[1], ctx.Args()[2])

	// Filter and run test cases.
	tests := suite.AllTests()
	if ctx.IsSet(testPatternFlag.Name) {
		tests = utesting.MatchTests(tests, ctx.String(testPatternFlag.Name))
	}
	results := utesting.RunTests(tests, os.Stdout)
	if fails := utesting.CountFailures(results); fails > 0 {
		return fmt.Errorf("%v of %v tests passed.", len(tests)-fails, len(tests))
	}
	fmt.Printf("all tests passed\n")
	return nil
}
