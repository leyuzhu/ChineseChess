package chess

import (
	"math/rand"
	"sort"
	"time"
)

//Search 与搜索有关的全局变量
type Search struct {
	mvResult      int        // 电脑走的棋
	nHistoryTable [65536]int // 历史表
}

//PositionStruct 局面结构
type PositionStruct struct {
	bFlipped    bool     //自己选择黑方
	sdPlayer    int      // 轮到谁走，0=红方，1=黑方
	vlRed       int      //红方的子力价值
	vlBlack     int      //黑方的子力价值
	nDistance   int      // 距离根节点的步数
	ucpcSquares [256]int // 棋盘上的棋子
	search      *Search
}

//NewPositionStruct 初始化棋局
func NewPositionStruct() *PositionStruct {
	p := &PositionStruct{
		search: &Search{},
	}
	if p == nil {
		return nil
	}
	return p
}

//startup 初始化棋盘
func (p *PositionStruct) startup() {
	pc := 0
	p.sdPlayer, p.vlRed, p.vlBlack, p.nDistance = 0, 0, 0, 0
	for i := 0; i < 256; i++ {
		p.ucpcSquares[i] = 0
	}
	for sq := 0; sq < 256; sq++ {
		pc = cucpcStartup[sq]
		if pc != 0 {
			p.addPiece(sq, pc)
		}
	}
}

//changeSide 交换走子方
func (p *PositionStruct) changeSide() {
	p.sdPlayer = 1 - p.sdPlayer
}

//addPiece 在棋盘上放一枚棋子
func (p *PositionStruct) addPiece(sq, pc int) {
	p.ucpcSquares[sq] = pc
	// 红方加分，黑方(注意"cucvlPiecePos"取值要颠倒)减分
	if pc < 16 {
		p.vlRed += cucvlPiecePos[pc-8][sq]
	} else {
		p.vlBlack += cucvlPiecePos[pc-16][squareFlip(sq)]
	}
}

//delPiece 从棋盘上拿走一枚棋子
func (p *PositionStruct) delPiece(sq, pc int) {
	p.ucpcSquares[sq] = 0
	// 红方减分，黑方(注意"cucvlPiecePos"取值要颠倒)加分
	if pc < 16 {
		p.vlRed -= cucvlPiecePos[pc-8][sq]
	} else {
		p.vlBlack -= cucvlPiecePos[pc-16][squareFlip(sq)]
	}
}

//evaluate 局面评价函数
func (p *PositionStruct) evaluate() int {
	if p.sdPlayer == 0 {
		return p.vlRed - p.vlBlack + AdvancedValue
	}

	return p.vlBlack - p.vlRed + AdvancedValue
}

//movePiece 搬一步棋的棋子
func (p *PositionStruct) movePiece(mv int) int {
	sqSrc := src(mv)
	sqDst := dst(mv)
	pcCaptured := p.ucpcSquares[sqDst]
	if pcCaptured != 0 {
		p.delPiece(sqDst, pcCaptured)
	}
	pc := p.ucpcSquares[sqSrc]
	p.delPiece(sqSrc, pc)
	p.addPiece(sqDst, pc)
	return pcCaptured
}

//undoMovePiece 撤消搬一步棋的棋子
func (p *PositionStruct) undoMovePiece(mv, pcCaptured int) {
	sqSrc := src(mv)
	sqDst := dst(mv)
	pc := p.ucpcSquares[sqDst]
	p.delPiece(sqDst, pc)
	p.addPiece(sqSrc, pc)
	if pcCaptured != 0 {
		p.addPiece(sqDst, pcCaptured)
	}
}

//makeMove 走一步棋
func (p *PositionStruct) makeMove(mv int) (bool, int) {
	pcCaptured := p.movePiece(mv)
	if p.checked() {
		p.undoMovePiece(mv, pcCaptured)
		return false, pcCaptured
	}
	p.changeSide()
	p.nDistance++
	return true, pcCaptured
}

//undoMakeMove 撤消走一步棋
func (p *PositionStruct) undoMakeMove(mv, pcCaptured int) {
	p.nDistance--
	p.changeSide()
	p.undoMovePiece(mv, pcCaptured)
}

//isMate 判断是否被将死
func (p *PositionStruct) isMate() bool {
	pcCaptured := 0
	mvs := make([]int, MaxGenMoves)
	nGenMoveNum := p.generateMoves(mvs)
	for i := 0; i < nGenMoveNum; i++ {
		pcCaptured = p.movePiece(mvs[i])
		if !p.checked() {
			p.undoMovePiece(mvs[i], pcCaptured)
			return false
		}

		p.undoMovePiece(mvs[i], pcCaptured)
	}
	return true
}

//flipBoard 反转棋盘
func (p *PositionStruct) flipBoard() {
	ucpcSquares := p.ucpcSquares
	for i := 0; i < 255; i++ { //255 not 256
		p.ucpcSquares[i] = ucpcSquares[squareFlip(i)]
	}
}

//legalMove 判断走法是否合理
func (p *PositionStruct) legalMove(mv int) bool {
	//判断起始格是否有自己的棋子
	sqSrc := src(mv)
	pcSrc := p.ucpcSquares[sqSrc]
	pcSelfSide := sideTag(p.sdPlayer)
	if (pcSrc & pcSelfSide) == 0 {
		return false
	}

	//判断目标格是否有自己的棋子
	sqDst := dst(mv)
	pcDst := p.ucpcSquares[sqDst]
	if (pcDst & pcSelfSide) != 0 {
		return false
	}

	//根据棋子的类型检查走法是否合理
	tmpPiece := pcSrc - pcSelfSide
	switch tmpPiece {
	case PieceJiang:
		return inFort(sqDst) && jiangSpan(sqSrc, sqDst)
	case PieceShi:
		return inFort(sqDst) && shiSpan(sqSrc, sqDst)
	case PieceXiang:
		return sameRiver(sqSrc, sqDst) && xiangSpan(sqSrc, sqDst) &&
			p.ucpcSquares[xiangPin(sqSrc, sqDst)] == 0
	case PieceMa:
		sqPin := maPin(sqSrc, sqDst)
		return sqPin != sqSrc && p.ucpcSquares[sqPin] == 0
	case PieceJu, PiecePao:
		nDelta := 0
		if sameX(sqSrc, sqDst) {
			if sqDst < sqSrc {
				nDelta = -1
			} else {
				nDelta = 1
			}
		} else if sameY(sqSrc, sqDst) {
			if sqDst < sqSrc {
				nDelta = -16
			} else {
				nDelta = 16
			}
		} else {
			return false
		}
		sqPin := sqSrc + nDelta
		for sqPin != sqDst && p.ucpcSquares[sqPin] == 0 {
			sqPin += nDelta
		}
		if sqPin == sqDst {
			return pcDst == 0 || tmpPiece == PieceJu
		} else if pcDst != 0 && tmpPiece == PiecePao {
			sqPin += nDelta
			for sqPin != sqDst && p.ucpcSquares[sqPin] == 0 {
				sqPin += nDelta
			}
			return sqPin == sqDst
		} else {
			return false
		}
	case PieceBing:
		if hasRiver(sqDst, p.sdPlayer, p.bFlipped) && (sqDst == sqSrc-1 || sqDst == sqSrc+1) {
			return true
		}
		return sqDst == squareForward(sqSrc, p.sdPlayer, p.bFlipped)
	default:
	}

	return false
}

//generateMoves 生成所有走法
func (p *PositionStruct) generateMoves(mvs []int) int {
	nGenMoves, pcSrc, sqDst, pcDst, nDelta := 0, 0, 0, 0, 0
	pcSelfSide := sideTag(p.sdPlayer)
	pcOppSide := oppSideTag(p.sdPlayer)

	for sqSrc := 0; sqSrc < 256; sqSrc++ {
		if !inBoard(sqSrc) {
			continue
		}
		// 找到一个本方棋子，再做以下判断：
		pcSrc = p.ucpcSquares[sqSrc]
		if (pcSrc & pcSelfSide) == 0 {
			continue
		}

		// 根据棋子确定走法
		switch pcSrc - pcSelfSide {
		case PieceJiang:
			for i := 0; i < 4; i++ {
				sqDst = sqSrc + ccJiangDelta[i]
				if !inFort(sqDst) {
					continue
				}
				pcDst = p.ucpcSquares[sqDst]
				if pcDst&pcSelfSide == 0 {
					mvs[nGenMoves] = move(sqSrc, sqDst)
					nGenMoves++
				}
			}
		case PieceShi:
			for i := 0; i < 4; i++ {
				sqDst = sqSrc + ccShiDelta[i]
				if !inFort(sqDst) {
					continue
				}
				pcDst = p.ucpcSquares[sqDst]
				if pcDst&pcSelfSide == 0 {
					mvs[nGenMoves] = move(sqSrc, sqDst)
					nGenMoves++
				}
			}
		case PieceXiang:
			for i := 0; i < 4; i++ {
				sqDst = sqSrc + ccShiDelta[i]
				if !(inBoard(sqDst) && noRiver(sqDst, p.sdPlayer, p.bFlipped) && p.ucpcSquares[sqDst] == 0) {
					continue
				}
				sqDst += ccShiDelta[i]
				pcDst = p.ucpcSquares[sqDst]
				if pcDst&pcSelfSide == 0 {
					mvs[nGenMoves] = move(sqSrc, sqDst)
					nGenMoves++
				}
			}
		case PieceMa:
			for i := 0; i < 4; i++ {
				sqDst = sqSrc + ccJiangDelta[i]
				if p.ucpcSquares[sqDst] != 0 {
					continue
				}
				for j := 0; j < 2; j++ {
					sqDst = sqSrc + ccMaDelta[i][j]
					if !inBoard(sqDst) {
						continue
					}
					pcDst = p.ucpcSquares[sqDst]
					if pcDst&pcSelfSide == 0 {
						mvs[nGenMoves] = move(sqSrc, sqDst)
						nGenMoves++
					}
				}
			}
		case PieceJu:
			for i := 0; i < 4; i++ {
				nDelta = ccJiangDelta[i]
				sqDst = sqSrc + nDelta
				for inBoard(sqDst) {
					pcDst = p.ucpcSquares[sqDst]
					if pcDst == 0 {
						mvs[nGenMoves] = move(sqSrc, sqDst)
						nGenMoves++
					} else {
						if (pcDst & pcOppSide) != 0 {
							mvs[nGenMoves] = move(sqSrc, sqDst)
							nGenMoves++
						}
						break
					}
					sqDst += nDelta
				}

			}
		case PiecePao:
			for i := 0; i < 4; i++ {
				nDelta = ccJiangDelta[i]
				sqDst = sqSrc + nDelta
				for inBoard(sqDst) {
					pcDst = p.ucpcSquares[sqDst]
					if pcDst == 0 {
						mvs[nGenMoves] = move(sqSrc, sqDst)
						nGenMoves++
					} else {
						break
					}
					sqDst += nDelta
				}
				sqDst += nDelta
				for inBoard(sqDst) {
					pcDst = p.ucpcSquares[sqDst]
					if pcDst != 0 {
						if (pcDst & pcOppSide) != 0 {
							mvs[nGenMoves] = move(sqSrc, sqDst)
							nGenMoves++
						}
						break
					}
					sqDst += nDelta
				}
			}
		case PieceBing:
			sqDst = squareForward(sqSrc, p.sdPlayer, p.bFlipped)
			if inBoard(sqDst) {
				pcDst = p.ucpcSquares[sqDst]
				if pcDst&pcSelfSide == 0 {
					mvs[nGenMoves] = move(sqSrc, sqDst)
					nGenMoves++
				}
			}
			if hasRiver(sqSrc, p.sdPlayer, p.bFlipped) {
				for nDelta = -1; nDelta <= 1; nDelta += 2 {
					sqDst = sqSrc + nDelta
					if inBoard(sqDst) {
						pcDst = p.ucpcSquares[sqDst]
						if pcDst&pcSelfSide == 0 {
							mvs[nGenMoves] = move(sqSrc, sqDst)
							nGenMoves++
						}
					}
				}
			}
		}
	}
	return nGenMoves
}

//checked 判断是否被将军
func (p *PositionStruct) checked() bool {
	nDelta, sqDst, pcDst := 0, 0, 0
	pcSelfSide := sideTag(p.sdPlayer)
	pcOppSide := oppSideTag(p.sdPlayer)

	for sqSrc := 0; sqSrc < 256; sqSrc++ {
		//找到棋盘上的帅(将)，再做以下判断：
		if !inBoard(sqSrc) || p.ucpcSquares[sqSrc] != pcSelfSide+PieceJiang {
			continue
		}

		//判断是否被对方的兵(卒)将军
		if p.ucpcSquares[squareForward(sqSrc, p.sdPlayer, p.bFlipped)] == pcOppSide+PieceBing {
			return true
		}
		for nDelta = -1; nDelta <= 1; nDelta += 2 {
			if p.ucpcSquares[sqSrc+nDelta] == pcOppSide+PieceBing {
				return true
			}
		}

		//判断是否被对方的马将军(以仕(士)的步长当作马腿)
		for i := 0; i < 4; i++ {
			if p.ucpcSquares[sqSrc+ccShiDelta[i]] != 0 {
				continue
			}
			for j := 0; j < 2; j++ {
				pcDst = p.ucpcSquares[sqSrc+ccMaCheckDelta[i][j]]
				if pcDst == pcOppSide+PieceMa {
					return true
				}
			}
		}

		//判断是否被对方的车或炮将军(包括将帅对脸)
		for i := 0; i < 4; i++ {
			nDelta = ccJiangDelta[i]
			sqDst = sqSrc + nDelta
			for inBoard(sqDst) {
				pcDst = p.ucpcSquares[sqDst]
				if pcDst != 0 {
					if pcDst == pcOppSide+PieceJu || pcDst == pcOppSide+PieceJiang {
						return true
					}
					break
				}
				sqDst += nDelta
			}
			sqDst += nDelta
			for inBoard(sqDst) {
				pcDst = p.ucpcSquares[sqDst]
				if pcDst != 0 {
					if pcDst == pcOppSide+PiecePao {
						return true
					}
					break
				}
				sqDst += nDelta
			}
		}
		return false
	}
	return false
}

//searchFull 超出边界(Fail-Soft)的Alpha-Beta搜索过程
func (p *PositionStruct) searchFull(vlAlpha, vlBeta, nDepth int) int {
	vl := 0

	//到达水平线，则返回局面评价值
	if nDepth <= 0 {
		return p.evaluate()
	}

	vlBest := -MateValue //是否一个走法都没走过(杀棋)
	mvBest := 0          //是否搜索到了Beta走法或PV走法，以便保存到历史表

	mvs := make([]int, MaxGenMoves)
	nGenMoves := p.generateMoves(mvs)
	mvs = mvs[:nGenMoves]
	sort.Slice(mvs, func(a, b int) bool {
		return p.search.nHistoryTable[a] > p.search.nHistoryTable[b]
	})

	//逐一走这些走法，并进行递归
	for i := 0; i < nGenMoves; i++ {
		if ok, pcCaptured := p.makeMove(mvs[i]); ok {
			vl = -p.searchFull(-vlBeta, -vlAlpha, nDepth-1)
			p.undoMakeMove(mvs[i], pcCaptured)

			//进行Alpha-Beta大小判断和截断
			if vl > vlBest {
				//找到最佳值(但不能确定是Alpha、PV还是Beta走法)
				vlBest = vl
				//vlBest就是目前要返回的最佳值，可能超出Alpha-Beta边界
				if vl >= vlBeta {
					//找到一个Beta走法, Beta走法要保存到历史表, 然后截断
					mvBest = mvs[i]
					break
				}
				if vl > vlAlpha {
					//找到一个PV走法，PV走法要保存到历史表，缩小Alpha-Beta边界
					mvBest = mvs[i]
					vlAlpha = vl
				}
			}
		}
	}

	//所有走法都搜索完了，把最佳走法(不能是Alpha走法)保存到历史表，返回最佳值
	if vlBest == -MateValue {
		//如果是杀棋，就根据杀棋步数给出评价
		return p.nDistance - MateValue
	}

	if mvBest != 0 {
		//如果不是Alpha走法，就将最佳走法保存到历史表
		p.search.nHistoryTable[mvBest] += nDepth * nDepth
		if p.nDistance == 0 {
			// 搜索根节点时，总是有一个最佳走法(因为全窗口搜索不会超出边界)，将这个走法保存下来
			p.search.mvResult = mvBest
		}
	}
	return vlBest
}

//searchMain 迭代加深搜索过程
func (p *PositionStruct) searchMain() {
	// 清空历史表
	for i := 0; i < 65536; i++ {
		p.search.nHistoryTable[i] = 0
	}

	// 初始化定时器
	start := time.Now()
	// 初始步数
	p.nDistance = 0

	// 迭代加深过程
	vl := 0
	rand.Seed(time.Now().UnixNano())
	for i := 1; i <= LimitDepth; i++ {
		vl = p.searchFull(-MateValue, MateValue, i)
		// 搜索到杀棋，就终止搜索
		if vl > WinValue || vl < -WinValue {
			break
		}
		// 超过一秒，就终止搜索
		if time.Now().Sub(start).Milliseconds() > 1000 {
			break
		}
	}
}
