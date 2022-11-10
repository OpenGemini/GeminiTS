package tokenMatchQuery

import (
	"github.com/openGemini/openGemini/lib/utils"
	"github.com/openGemini/openGemini/lib/vToken/tokenDic/tokenClvc"
	"github.com/openGemini/openGemini/lib/vToken/tokenIndex"
	"sort"
)

func MatchSearch(searchStr string, root *tokenClvc.TrieTreeNode, indexRoot *tokenIndex.IndexTreeNode, qmin int) []utils.SeriesId {
	var vgMap = make(map[uint16][]string)
	searchtoken, _ := utils.DataProcess(searchStr)
	tokenIndex.VGCons(root, qmin, searchtoken, vgMap)
	//fmt.Println(vgMap)

	var sortSumInvertList = make([]SortKey, 0)
	for x := range vgMap {
		token := vgMap[x]
		if token != nil {
			var invertIndex tokenIndex.Inverted_index
			var indexNode *tokenIndex.IndexTreeNode
			var invertIndex1 tokenIndex.Inverted_index
			var invertIndex2 tokenIndex.Inverted_index
			var invertIndex3 tokenIndex.Inverted_index
			invertIndex1, indexNode = SearchInvertedListFromCurrentNode(token, indexRoot, 0, invertIndex1, indexNode)
			invertIndex = DeepCopy(invertIndex1)
			invertIndex2 = SearchInvertedListFromChildrensOfCurrentNode(indexNode, nil)
			if indexNode != nil && len(indexNode.AddrOffset()) > 0 {
				invertIndex3 = TurnAddr2InvertLists(indexNode.AddrOffset(), invertIndex3)
			}
			invertIndex = MergeMapsInvertLists(invertIndex2, invertIndex)
			invertIndex = MergeMapsInvertLists(invertIndex3, invertIndex)
			sortSumInvertList = append(sortSumInvertList, NewSortKey(x, len(invertIndex), token, invertIndex))
		}
	}
	sort.SliceStable(sortSumInvertList, func(i, j int) bool {
		if sortSumInvertList[i].sizeOfInvertedList < sortSumInvertList[j].sizeOfInvertedList {
			return true
		}
		return false
	})

	var resArr = make([]utils.SeriesId, 0)
	var preSeaPosition uint16 = 0
	var preInverPositionDis = make([]PosList, 0)
	var nowInverPositionDis = make([]PosList, 0)
	for m := 0; m < len(sortSumInvertList); m++ {
		tokenArr := sortSumInvertList[m].tokenArr
		var nowSeaPosition uint16
		if tokenArr != nil {
			nowSeaPosition = sortSumInvertList[m].offset
			var invertIndex tokenIndex.Inverted_index = nil
			invertIndex = sortSumInvertList[m].invertedIndex
			//fmt.Println(len(invertIndex))
			if invertIndex == nil {
				return nil
			}
			if m == 0 {
				for sid := range invertIndex {
					preInverPositionDis = append(preInverPositionDis, NewPosList(sid, make([]uint16, len(invertIndex[sid]), len(invertIndex[sid]))))
					nowInverPositionDis = append(nowInverPositionDis, NewPosList(sid, invertIndex[sid]))
					resArr = append(resArr, sid)
				}
			} else {
				for j := 0; j < len(resArr); j++ { //遍历之前合并好的resArr
					findFlag := false
					sid := resArr[j]
					if _, ok := invertIndex[sid]; ok {
						nowInverPositionDis[j] = NewPosList(sid, invertIndex[sid])
						for z1 := 0; z1 < len(preInverPositionDis[j].posArray); z1++ {
							z1Pos := preInverPositionDis[j].posArray[z1]
							for z2 := 0; z2 < len(nowInverPositionDis[j].posArray); z2++ {
								z2Pos := nowInverPositionDis[j].posArray[z2]
								if nowSeaPosition-preSeaPosition == z2Pos-z1Pos {
									findFlag = true
									break
								}
							}
							if findFlag == true {
								break
							}
						}
					}
					if findFlag == false { //没找到并且候选集的sid比resArr大，删除resArr[j]
						resArr = append(resArr[:j], resArr[j+1:]...)
						preInverPositionDis = append(preInverPositionDis[:j], preInverPositionDis[j+1:]...)
						nowInverPositionDis = append(nowInverPositionDis[:j], nowInverPositionDis[j+1:]...)
						j-- //删除后重新指向，防止丢失元素判断
					}
				}
			}
			preSeaPosition = nowSeaPosition
			copy(preInverPositionDis, nowInverPositionDis)
		}
	}
	sort.SliceStable(resArr, func(i, j int) bool {
		if resArr[i].Id < resArr[j].Id && resArr[i].Time < resArr[j].Time {
			return true
		}
		return false
	})
	return resArr
}

func SearchInvertedListFromCurrentNode(tokenArr []string, indexRoot *tokenIndex.IndexTreeNode, i int, invertIndex1 tokenIndex.Inverted_index, indexNode *tokenIndex.IndexTreeNode) (tokenIndex.Inverted_index, *tokenIndex.IndexTreeNode) {
	if indexRoot == nil {
		return invertIndex1, indexNode
	}
	if i < len(tokenArr)-1 && indexRoot.Children()[utils.StringToHashCode(tokenArr[i])] != nil {
		invertIndex1, indexNode = SearchInvertedListFromCurrentNode(tokenArr, indexRoot.Children()[utils.StringToHashCode(tokenArr[i])], i+1, invertIndex1, indexNode)
	}
	if i == len(tokenArr)-1 && indexRoot.Children()[utils.StringToHashCode(tokenArr[i])] != nil { //找到那一层的倒排表
		invertIndex1 = indexRoot.Children()[utils.StringToHashCode(tokenArr[i])].InvertedIndex()
		indexNode = indexRoot.Children()[utils.StringToHashCode(tokenArr[i])]
	}
	return invertIndex1, indexNode
}

func SearchInvertedListFromChildrensOfCurrentNode(indexNode *tokenIndex.IndexTreeNode, invertIndex2 tokenIndex.Inverted_index) tokenIndex.Inverted_index {
	if indexNode != nil {
		for _, child := range indexNode.Children() {
			if len(child.InvertedIndex()) > 0 {
				invertIndex2 = MergeMapsInvertLists(child.InvertedIndex(), invertIndex2)
			}
			if len(child.AddrOffset()) > 0 {
				var invertIndex3 = TurnAddr2InvertLists(child.AddrOffset(), nil)
				invertIndex2 = MergeMapsInvertLists(invertIndex3, invertIndex2)
			}
			invertIndex2 = SearchInvertedListFromChildrensOfCurrentNode(child, invertIndex2)
		}
	}
	return invertIndex2
}

func TurnAddr2InvertLists(addrOffset map[*tokenIndex.IndexTreeNode]uint16, invertIndex3 tokenIndex.Inverted_index) tokenIndex.Inverted_index {
	var res tokenIndex.Inverted_index
	for addr, offset := range addrOffset {
		invertIndex3 = make(map[utils.SeriesId][]uint16)
		for key, value := range addr.InvertedIndex() {
			list := make([]uint16, 0)
			for i := 0; i < len(value); i++ {
				list = append(list, value[i]+offset)
			}
			invertIndex3[key] = list
		}
		res = MergeMapsTwoInvertLists(invertIndex3, res)
	}
	return res
}

func MergeMapsInvertLists(map1 map[utils.SeriesId][]uint16, map2 map[utils.SeriesId][]uint16) map[utils.SeriesId][]uint16 {
	if len(map2) > 0 {
		for sid1, list1 := range map1 {
			if list2, ok := map2[sid1]; !ok {
				map2[sid1] = list1
			} else {
				list2 = append(list2, list1...)
				list2 = UniqueArr(list2)
				sort.Slice(list2, func(i, j int) bool { return list2[i] < list2[j] })
				map2[sid1] = list2
			}
		}
	} else {
		map2 = DeepCopy(map1)
	}
	return map2
}

func UniqueArr(m []uint16) []uint16 {
	d := make([]uint16, 0)
	tempMap := make(map[uint16]bool, len(m))
	for _, v := range m { // 以值作为键名
		if tempMap[v] == false {
			tempMap[v] = true
			d = append(d, v)
		}
	}
	return d
}

func DeepCopy(src map[utils.SeriesId][]uint16) map[utils.SeriesId][]uint16 {
	dst := make(map[utils.SeriesId][]uint16)
	for key, value := range src {
		list := make([]uint16, 0)
		for i := 0; i < len(value); i++ {
			list = append(list, value[i])
		}
		dst[key] = list
	}
	return dst
}

func MergeMapsTwoInvertLists(map1 map[utils.SeriesId][]uint16, map2 map[utils.SeriesId][]uint16) map[utils.SeriesId][]uint16 {
	if len(map1) == 0 {
		return map2
	} else if len(map2) == 0 {
		return map1
	} else if len(map1) < len(map2) {
		for sid1, list1 := range map1 {
			if list2, ok := map2[sid1]; !ok {
				map2[sid1] = list1
			} else {
				list2 = append(list2, list1...)
				list2 = UniqueArr(list2)
				sort.Slice(list2, func(i, j int) bool { return list2[i] < list2[j] })
				map2[sid1] = list2
			}
		}
		return map2
	} else {
		for sid1, list1 := range map2 {
			if list2, ok := map1[sid1]; !ok {
				map1[sid1] = list1
			} else {
				list2 = append(list2, list1...)
				list2 = UniqueArr(list2)
				sort.Slice(list2, func(i, j int) bool { return list2[i] < list2[j] })
				map1[sid1] = list2
			}
		}
		return map1
	}
}