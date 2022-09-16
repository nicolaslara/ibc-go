package transfer

import (
	"context"
	"testing"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	"github.com/strangelove-ventures/ibctest/test"
)

// TestMsgTransfer_WithAndWithoutMetadata will test sending successful IBC transfers
// from chainA to chainB and back.
// If the chains contain a version of FungibleTokenPacketData with metadata, both sends should succeed.
// If one of the chains contains a version of FungibleTokenPacketData without metadata, then receiving a packet with
// metadata should fail in that chain
func (s *TransferTestSuite) TestMsgTransfer_WithAndWithoutMetadata() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	chainAVersion := chainA.Config().Images[0].Version
	chainBVersion := chainB.Config().Images[0].Version
	t.Logf("Running metadata tests versions chainA: %s, chainB: %s", chainAVersion, chainBVersion)

	t.Run("IBC token transfer with metadata from chainA to chainB", func(t *testing.T) {
		// this should only pass if both chains have the metadata as part of the message

		// Right now this works when there is no metadata, but fails when there is
		// (even if the both chains are in the latest version)
		// Need to get this to work so that we can check all the combinations of (with/without metadata  x  old/new chain version)
		transferTxResp, err := s.TransferWithMetadata(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, []byte(""))
		s.Require().NoError(err)
		s.AssertValidTxResponse(transferTxResp)

		// Experiment sending via command line (but can't easily add the metadata here because the params are not exposed)
		//amount := ibc.WalletAmount{
		//	Amount:  testvalues.IBCTransferAmount,
		//	Denom:   chainADenom,
		//	Address: chainAAddress,
		//}
		//chainATx, err := chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName, amount, nil)
		//s.Require().NoError(err)
		//s.Require().NoError(chainATx.Validate(), "chain-a ibc transfer tx is invalid")
		//

	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("non-native IBC token transfer from chainB to chainA, receiver is source of tokens", func(t *testing.T) {
		transferTxResp, err := s.TransferWithMetadata(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, []byte{})
		s.Require().NoError(err)
		s.AssertValidTxResponse(transferTxResp)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
	})
}
