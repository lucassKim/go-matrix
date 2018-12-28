// Copyright (c) 2018 The MATRIX Authors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php
package layered

import (
	//"fmt"

	"github.com/matrix/go-matrix/baseinterface"
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core/vm"
	"github.com/matrix/go-matrix/election/support"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/mc"
)

type Echelon struct {
	MinMoney uint64
	Quota    int
}

const (
	DefauleStock = 1
)

var (
	FirstEchelon = Echelon{
		MinMoney: 10000000,
		Quota:    3,
	}
	SecondEchelon = Echelon{
		MinMoney: 100000,
		Quota:    3,
	}
)

type layered struct {
}

func init() {
	baseinterface.RegElectPlug("layered", RegInit)
}

func RegInit() baseinterface.ElectionInterface {
	return &layered{}
}

func (self *layered) MinerTopGen(mmrerm *mc.MasterMinerReElectionReqMsg) *mc.MasterMinerReElectionRsp {
	log.INFO("分层方案", "矿工拓扑生成", len(mmrerm.MinerList))
	return support.MinerTopGen(mmrerm)

}

func (self *layered) ValidatorTopGen(mvrerm *mc.MasterValidatorReElectionReqMsg) *mc.MasterValidatorReElectionRsq {
	log.INFO("分层方案", "验证者拓扑生成", len(mvrerm.ValidatorList))
	ValidatorTopGen := mc.MasterValidatorReElectionRsq{}
	ChoiceToMaster := make(map[common.Address]int, 0)

	InitMapList := make(map[string]vm.DepositDetail, 0)

	for _, v := range mvrerm.ValidatorList {
		InitMapList[v.NodeID.String()] = v
	}

	FirstQuota, SecondQuota := CalEchelonNum(mvrerm.ValidatorList)
	//fmt.Println(len(FirstQuota), len(SecondQuota))

	if len(FirstQuota) > FirstEchelon.Quota {
		FirstQuota = sortByDepositAndUptime(FirstQuota)
	}
	for _, v := range FirstQuota {
		tempNodeInfo := mc.TopologyNodeInfo{
			Account:  v.Address,
			Position: uint16(len(ValidatorTopGen.MasterValidator)),
			Stock:    DefauleStock,
			Type:     common.RoleValidator,
		}
		ValidatorTopGen.MasterValidator = append(ValidatorTopGen.MasterValidator, tempNodeInfo)
		ChoiceToMaster[v.Address] = 1
		if len(ValidatorTopGen.MasterValidator) >= FirstEchelon.Quota {
			break
		}
	}

	if len(SecondQuota) > SecondEchelon.Quota {
		SecondQuota = sortByDepositAndUptime(SecondQuota)
	}
	for _, v := range SecondQuota {
		tempNodeInfo := mc.TopologyNodeInfo{
			Account:  v.Address,
			Position: uint16(len(ValidatorTopGen.MasterValidator)),
			Stock:    DefauleStock,
			Type:     common.RoleValidator,
		}
		ValidatorTopGen.MasterValidator = append(ValidatorTopGen.MasterValidator, tempNodeInfo)
		ChoiceToMaster[v.Address] = 1
		if len(ValidatorTopGen.MasterValidator) >= SecondEchelon.Quota+FirstEchelon.Quota {
			break
		}
	}
	//fmt.Println("94", len(ValidatorTopGen.MasterValidator), len(ValidatorTopGen.BackUpValidator))

	NowList := []vm.DepositDetail{}
	for _, v := range mvrerm.ValidatorList {
		_, ok := ChoiceToMaster[v.Address]
		if ok {
			continue
		}
		NowList = append(NowList, v)
	}
	weight := GetValueByDeposit(NowList)
	//fmt.Println("weight", len(weight))
	//fmt.Println("zzz", support.M-len(ValidatorTopGen.MasterValidator))

	a, b, c := support.ValNodesSelected(weight, mvrerm.RandSeed.Int64(), support.M-len(ValidatorTopGen.MasterValidator), 5, 0) //mvrerm.RandSeed.Int64(), 11, 5, 0) //0x12217)

	//fmt.Println(len(a), len(b), len(c))
	for _, v := range a {
		tempNodeInfo := mc.TopologyNodeInfo{
			Account:  InitMapList[v.Nodeid].Address,
			Position: uint16(len(ValidatorTopGen.MasterValidator)),
			Stock:    DefauleStock,
			Type:     common.RoleValidator,
		}
		ValidatorTopGen.MasterValidator = append(ValidatorTopGen.MasterValidator, tempNodeInfo)
	}
	for _, v := range b {
		tempNodeInfo := mc.TopologyNodeInfo{
			Account:  InitMapList[v.Nodeid].Address,
			Position: uint16(len(ValidatorTopGen.BackUpValidator)),
			Stock:    DefauleStock,
			Type:     common.RoleBackupValidator,
		}
		ValidatorTopGen.BackUpValidator = append(ValidatorTopGen.BackUpValidator, tempNodeInfo)
	}
	for _, v := range c {
		tempNodeInfo := mc.TopologyNodeInfo{
			Account:  InitMapList[v.Nodeid].Address,
			Position: uint16(len(ValidatorTopGen.CandidateValidator)),
			Stock:    DefauleStock,
			Type:     common.RoleCandidateValidator,
		}
		ValidatorTopGen.CandidateValidator = append(ValidatorTopGen.CandidateValidator, tempNodeInfo)
	}

	return &ValidatorTopGen

}

func (self *layered) ToPoUpdate(Q0, Q1, Q2 []mc.TopologyNodeInfo, nettopo mc.TopologyGraph, offline []common.Address) []mc.Alternative {
	return support.ToPoUpdate(Q0, Q1, Q2, nettopo, offline)
}

func (self *layered) PrimarylistUpdate(Q0, Q1, Q2 []mc.TopologyNodeInfo, online mc.TopologyNodeInfo, flag int) ([]mc.TopologyNodeInfo, []mc.TopologyNodeInfo, []mc.TopologyNodeInfo) {
	return support.PrimarylistUpdate(Q0, Q1, Q2, online, flag)
}