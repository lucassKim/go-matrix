//1543547730.107878
//1543547062.7631512
//1543546400.0274198
//1543545592.1725566
//1543544923.3565996
//1543544176.662785
// Copyright (c) 2018 The MATRIX Authors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php

package leaderelect

import (
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/mc"
	"github.com/matrix/go-matrix/params/manparams"
	"github.com/pkg/errors"
)

type leaderCalculator struct {
	preLeader   common.Address
	preHash     common.Hash
	preIsSupper bool
	leaderList  map[uint32]common.Address
	validators  []mc.TopologyNodeInfo
	chain       *core.BlockChain
	cdc         *cdc
}

func newLeaderCalculator(chain *core.BlockChain, cdc *cdc) *leaderCalculator {
	return &leaderCalculator{
		preLeader:   common.Address{},
		preHash:     common.Hash{},
		preIsSupper: false,
		leaderList:  make(map[uint32]common.Address),
		validators:  nil,
		chain:       chain,
		cdc:         cdc,
	}
}

func (self *leaderCalculator) SetValidators(preHash common.Hash, preIsSupper bool, preLeader common.Address, validators []mc.TopologyNodeInfo) error {
	if validators == nil {
		return ErrValidatorsIsNil
	}

	preNumber := self.cdc.number - 1
	if preIsSupper == false && common.IsBroadcastNumber(preNumber) && preNumber != 0 {
		header := self.chain.GetHeaderByNumber(preNumber - 1)
		if nil == header {
			log.ERROR("")
			return errors.Errorf("获取广播区块前一区块(%d)错误!", preNumber-1)
		}
		preLeader = header.Leader
	}
	log.INFO(self.cdc.logInfo, "计算leader列表", "开始", "preLeader", preLeader.Hex(), "前一个区块是否为超级区块", preIsSupper)
	leaderList, err := calLeaderList(preLeader, preNumber, preIsSupper, validators)
	if err != nil {
		return err
	}
	self.leaderList = leaderList
	self.preLeader.Set(preLeader)
	self.preHash.Set(preHash)
	self.validators = validators
	self.preIsSupper = preIsSupper

	return nil
}

func (self *leaderCalculator) GetValidators() (*mc.TopologyGraph, error) {
	if len(self.validators) == 0 {
		return nil, errors.New("验证者列表为空")
	}
	rlt := &mc.TopologyGraph{}
	for i := 0; i < len(self.validators); i++ {
		rlt.NodeList = append(rlt.NodeList, self.validators[i])
	}
	return rlt, nil
}

func (self *leaderCalculator) GetLeader(turn uint32) (*leaderData, error) {
	leaderCount := uint32(len(self.leaderList))
	if leaderCount == 0 {
		return nil, ErrValidatorsIsNil
	}

	leaders := &leaderData{}
	number := self.cdc.number
	if common.IsReElectionNumber(number) {
		leaders.leader.Set(manparams.BroadCastNodes[0].Address)
		leaders.nextLeader.Set(self.leaderList[turn%leaderCount])
		return leaders, nil
	}

	if common.IsBroadcastNumber(number) {
		leaders.leader.Set(manparams.BroadCastNodes[0].Address)
		leaders.nextLeader.Set(self.leaderList[(turn)%leaderCount])
		return leaders, nil
	}

	leaders.leader.Set(self.leaderList[turn%leaderCount])
	if common.IsBroadcastNumber(number + 1) {
		leaders.nextLeader.Set(manparams.BroadCastNodes[0].Address)
	} else {
		leaders.nextLeader.Set(self.leaderList[(turn+1)%leaderCount])
	}
	return leaders, nil
}

func calLeaderList(preLeader common.Address, preNumber uint64, preIsSupper bool, validators []mc.TopologyNodeInfo) (map[uint32]common.Address, error) {
	ValidatorNum := len(validators)
	var startPos = 0
	if preIsSupper || common.IsReElectionNumber(preNumber) || common.IsReElectionNumber(preNumber+1) {
		startPos = 0
	} else {
		preIndex, err := findLeaderIndex(preLeader, validators)
		if err != nil {
			return nil, err
		}
		startPos = preIndex + 1
	}
	leaderList := make(map[uint32]common.Address)
	for i := 0; i < ValidatorNum; i++ {
		leaderList[uint32(i)] = validators[(startPos+int(i))%ValidatorNum].Account
	}
	return leaderList, nil
}

func findLeaderIndex(preLeader common.Address, validators []mc.TopologyNodeInfo) (int, error) {
	for index, v := range validators {
		if v.Account == preLeader {
			return index, nil
		}
	}
	return 0, ErrValidatorNotFound
}
