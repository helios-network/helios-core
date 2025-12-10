// Copyright Jeremy Guyet

package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/gogoproto/proto"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	cmn "helios-core/helios-chain/precompiles/common"
	rpctypes "helios-core/helios-chain/rpc/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	logostypes "helios-core/helios-chain/x/logos/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func ParseProposal(proposal *govtypes.Proposal, govParams *govtypes.Params, codec codec.Codec) (*rpctypes.ProposalRPC, error) {
	statusTypes := map[govtypes.ProposalStatus]string{
		govtypes.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED:    "UNSPECIFIED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD: "DEPOSIT_PERIOD",
		govtypes.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD:  "VOTING_PERIOD",
		govtypes.ProposalStatus_PROPOSAL_STATUS_PASSED:         "PASSED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_REJECTED:       "REJECTED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_FAILED:         "FAILED",
	}

	proposerAddr, err := sdk.AccAddressFromBech32(proposal.Proposer)
	if err != nil {
		return nil, err
	}
	details := make([]map[string]interface{}, 0)

	for _, anyJSON := range proposal.Messages {
		msg := &govtypes.MsgExecLegacyContent{}

		err := proto.Unmarshal(anyJSON.Value, msg)
		if err != nil {
			details = append(details, map[string]interface{}{
				"type":  "UnknownProposalType",
				"error": err.Error(),
			})
			continue
		}

		contentJson, err := codec.MarshalInterfaceJSON(msg)
		if err != nil {
			details = append(details, map[string]interface{}{
				"type":  msg.Content.TypeUrl,
				"error": err.Error(),
			})
			continue
		}
		// json to interface
		var content map[string]interface{}
		err = json.Unmarshal(contentJson, &content)
		if err != nil {
			details = append(details, map[string]interface{}{
				"type":  msg.Content.TypeUrl,
				"error": err.Error(),
			})
			continue
		}
		decodedContent := content["content"].(map[string]interface{})

		// check if msg field exists and is string
		if decodedContent["msg"] != nil && decodedContent["msg"].(string) != "" {
			var interfaceMsgMap map[string]interface{}
			err = json.Unmarshal([]byte(decodedContent["msg"].(string)), &interfaceMsgMap)
			if err != nil {
				details = append(details, map[string]interface{}{
					"type":  msg.Content.TypeUrl,
					"error": err.Error(),
				})
				continue
			}
			decodedContent["msg"] = interfaceMsgMap
		}

		details = append(details, content["content"].(map[string]interface{}))
	}

	// return map[string]interface{}{
	// 	"id":         proposal.Id,
	// 	"statusCode": proposal.Status,
	// 	"status":     statusTypes[proposal.Status],
	// 	"proposer":   common.BytesToAddress(proposerAddr.Bytes()).String(),
	// 	"title":      proposal.Title,
	// 	"metadata":   proposal.Metadata,
	// 	"summary":    proposal.Summary,
	// 	"details":    details,
	// 	"options": []*govtypes.WeightedVoteOption{
	// 		{Option: govtypes.OptionYes, Weight: "Yes"},
	// 		{Option: govtypes.OptionAbstain, Weight: "Abstain"},
	// 		{Option: govtypes.OptionNo, Weight: "No"},
	// 		{Option: govtypes.OptionNoWithVeto, Weight: "No With Veto"},
	// 	},
	// 	"votingStartTime":    proposal.VotingStartTime,
	// 	"votingEndTime":      proposal.VotingEndTime,
	// 	"submitTime":         proposal.SubmitTime,
	// 	"totalDeposit":       proposal.TotalDeposit,
	// 	"minDeposit":         proposal.GetMinDepositFromParams(*govParams),
	// 	"finalTallyResult":   proposal.FinalTallyResult,
	// 	"currentTallyResult": proposal.CurrentTallyResult,
	// }, nil

	return &rpctypes.ProposalRPC{
		Id:         proposal.Id,
		StatusCode: proposal.Status.String(),
		Status:     statusTypes[proposal.Status],
		Proposer:   common.BytesToAddress(proposerAddr.Bytes()).String(),
		Title:      proposal.Title,
		Metadata:   proposal.Metadata,
		Summary:    proposal.Summary,
		Details:    details,
		Options: []rpctypes.ProposalVoteOptionRPC{
			{Option: govtypes.OptionYes.String(), Weight: "Yes"},
			{Option: govtypes.OptionAbstain.String(), Weight: "Abstain"},
			{Option: govtypes.OptionNo.String(), Weight: "No"},
			{Option: govtypes.OptionNoWithVeto.String(), Weight: "No With Veto"},
		},
		VotingStartTime:    *proposal.VotingStartTime,
		VotingEndTime:      *proposal.VotingEndTime,
		SubmitTime:         *proposal.SubmitTime,
		TotalDeposit:       proposal.TotalDeposit,
		MinDeposit:         proposal.GetMinDepositFromParams(*govParams),
		FinalTallyResult:   *proposal.FinalTallyResult,
		CurrentTallyResult: *proposal.CurrentTallyResult,
	}, nil
}

func (b *Backend) GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalRPC, error) {
	proposalsResult := make([]*rpctypes.ProposalRPC, 0)
	proposals, err := b.queryClient.Gov.Proposals(b.ctx, &govtypes.QueryProposalsRequest{
		Pagination: &query.PageRequest{
			Offset:  (uint64(page) - 1) * uint64(size),
			Limit:   uint64(size),
			Reverse: true,
		},
	})
	if err != nil {
		return nil, err
	}

	msg := &govtypes.QueryParamsRequest{
		ParamsType: "voting",
	}
	resParams, err := b.queryClient.Gov.Params(b.ctx, msg)
	if err != nil {
		return nil, err
	}
	for _, proposal := range proposals.Proposals {
		formattedProposal, err := ParseProposal(proposal, resParams.Params, b.clientCtx.Codec)
		if err != nil {
			continue
		}
		proposalsResult = append(proposalsResult, formattedProposal)
	}
	return proposalsResult, nil
}

func (b *Backend) GetProposal(id hexutil.Uint64) (*rpctypes.ProposalRPC, error) {
	proposalResponse, err := b.queryClient.Gov.Proposal(b.ctx, &govtypes.QueryProposalRequest{
		ProposalId: uint64(id),
	})
	if err != nil {
		return nil, err
	}
	msg := &govtypes.QueryParamsRequest{
		ParamsType: "voting",
	}
	resParams, err := b.queryClient.Gov.Params(b.ctx, msg)
	if err != nil {
		return nil, err
	}
	formattedProposal, err := ParseProposal(proposalResponse.Proposal, resParams.Params, b.clientCtx.Codec)
	if err != nil {
		return nil, err
	}
	return formattedProposal, nil
}

func (b *Backend) GetProposalVotesByPageAndSize(id uint64, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalVoteRPC, error) {
	proposalResponse, err := b.queryClient.Gov.Proposal(b.ctx, &govtypes.QueryProposalRequest{
		ProposalId: uint64(id),
	})
	if err != nil {
		return nil, err
	}
	if proposalResponse.Proposal == nil {
		return nil, errors.New("proposal not found")
	}
	proposalVotes, err := b.queryClient.Gov.Votes(b.ctx, &govtypes.QueryVotesRequest{
		ProposalId: uint64(id),
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
	})
	if err != nil {
		return nil, err
	}
	proposalVotesResult := make([]*rpctypes.ProposalVoteRPC, 0)
	for _, vote := range proposalVotes.Votes {
		options := make([]rpctypes.ProposalVoteOptionRPC, 0)
		for _, option := range vote.Options {
			options = append(options, rpctypes.ProposalVoteOptionRPC{
				Option: option.Option.String(),
				Weight: option.Weight,
			})
		}
		proposalVotesResult = append(proposalVotesResult, &rpctypes.ProposalVoteRPC{
			Voter:    cmn.AnyToHexAddress(vote.Voter).Hex(),
			Options:  options,
			Metadata: vote.Metadata,
		})
	}
	return proposalVotesResult, nil
}

func (b *Backend) GetProposalsCount() (*hexutil.Uint64, error) {
	response, err := b.queryClient.Gov.ProposalsCount(b.ctx, &govtypes.QueryProposalsCountRequest{})
	if err != nil {
		return nil, err
	}
	totalCount := hexutil.Uint64(response.Count)
	return &totalCount, nil
}

type ProposalFilter struct {
	Status      govtypes.ProposalStatus
	Proposer    string
	Title       string
	Description string
	Voter       string
	Depositor   string
	Request     *govtypes.QueryProposalsRequest
}

func (b *Backend) GetProposalsByPageAndSizeWithFilter(page hexutil.Uint64, size hexutil.Uint64, filter string) ([]*rpctypes.ProposalRPC, error) {
	// filters examples:
	// status=1
	// status=2
	// status=3
	// proposer=0xffffffffffffffffffffffffffffffffffffffff
	// title-matches=test
	// description-matches=test
	// status=1&proposer=0xffffffffffffffffffffffffffffffffffffffff&title-matches=test&description-matches=test

	var proposalFilter ProposalFilter
	proposalFilter.Status = govtypes.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED
	proposalFilter.Proposer = ""
	proposalFilter.Voter = ""
	proposalFilter.Depositor = ""
	proposalFilter.Title = ""
	proposalFilter.Description = ""
	proposalFilter.Request = &govtypes.QueryProposalsRequest{
		Pagination: &query.PageRequest{
			Offset:  (uint64(page) - 1) * uint64(size),
			Limit:   uint64(size),
			Reverse: true,
		},
	}

	for _, filter := range strings.Split(filter, "&") {
		parts := strings.Split(filter, "=")
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "status":
			parsedStatus, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				continue
			}
			proposalFilter.Status = govtypes.ProposalStatus(uint32(parsedStatus))
			proposalFilter.Request.ProposalStatus = proposalFilter.Status
		case "proposer":
			proposalFilter.Proposer = cmn.AccAddressFromHexAddress(cmn.AnyToHexAddress(parts[1])).String()
			proposalFilter.Request.Proposer = proposalFilter.Proposer
		case "depositor":
			proposalFilter.Depositor = cmn.AccAddressFromHexAddress(cmn.AnyToHexAddress(parts[1])).String()
			proposalFilter.Request.Depositor = proposalFilter.Depositor
		case "voter":
			proposalFilter.Voter = cmn.AccAddressFromHexAddress(cmn.AnyToHexAddress(parts[1])).String()
			proposalFilter.Request.Voter = proposalFilter.Voter
		case "title-matches":
			proposalFilter.Title = parts[1]
		case "description-matches":
			proposalFilter.Description = parts[1]
		}
	}

	proposalsResult := make([]*rpctypes.ProposalRPC, 0)
	proposals, err := b.queryClient.Gov.Proposals(b.ctx, proposalFilter.Request)
	if err != nil {
		return nil, err
	}

	msg := &govtypes.QueryParamsRequest{
		ParamsType: "voting",
	}
	resParams, err := b.queryClient.Gov.Params(b.ctx, msg)
	if err != nil {
		return nil, err
	}
	for _, proposal := range proposals.Proposals {
		formattedProposal, err := ParseProposal(proposal, resParams.Params, b.clientCtx.Codec)
		if err != nil {
			continue
		}
		proposalsResult = append(proposalsResult, formattedProposal)
	}
	return proposalsResult, nil
}

func InferModuleFromTypeURL(typeURL string) string {
	// "/cosmos.slashing.v1beta1.MsgUpdateParams" -> "slashing"
	s := strings.TrimPrefix(typeURL, "/")
	parts := strings.Split(s, ".")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (b *Backend) MessageRequiresAuthority(protoFullName string) (bool, error) {
	// normaliser : enlever le slash initial si présent
	protoFullNameWithoutSlash := strings.TrimPrefix(protoFullName, "/")

	// 1) try GlobalTypes first
	if mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(protoFullNameWithoutSlash)); err == nil {
		return mdHasAuthority(mt.Descriptor()), nil
	}

	// 2) fallback: search in GlobalFiles
	md, err := findMessageDescriptorInFiles(protoFullNameWithoutSlash)
	if err != nil {
		return false, err
	}
	return mdHasAuthority(md), nil
}

func findMessageDescriptorInFiles(fullName string) (protoreflect.MessageDescriptor, error) {
	var found protoreflect.MessageDescriptor
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		// iterate top-level messages
		for i := 0; i < fd.Messages().Len(); i++ {
			if md := searchMessageRec(fd.Messages().Get(i), fullName); md != nil {
				found = md
				return false // stop iteration early
			}
		}
		return true
	})
	if found == nil {
		return nil, fmt.Errorf("message %s not found in protoregistry.GlobalFiles", fullName)
	}
	return found, nil
}

func searchMessageRec(md protoreflect.MessageDescriptor, fullName string) protoreflect.MessageDescriptor {
	if string(md.FullName()) == fullName {
		return md
	}
	// nested messages
	nested := md.Messages()
	for i := 0; i < nested.Len(); i++ {
		if res := searchMessageRec(nested.Get(i), fullName); res != nil {
			return res
		}
	}
	return nil
}

// GovCatalog construit et retourne le catalogue (tu peux le cacher après le build)
func (b *Backend) GovCatalog() ([]rpctypes.MsgCatalogEntry, error) {
	// récupère toutes les impls de Msg (type_url list)
	impls := b.clientCtx.InterfaceRegistry.ListImplementations("cosmos.base.v1beta1.Msg")

	var out []rpctypes.MsgCatalogEntry
	for _, typeURL := range impls {
		// normalise: " /cosmos.bank.v1beta1.MsgSend" -> "cosmos.bank.v1beta1.MsgSend"
		protoName := strings.TrimPrefix(typeURL, "/")

		// récupère descriptor via InterfaceRegistry
		desc, err := b.clientCtx.Codec.InterfaceRegistry().FindDescriptorByName(protoreflect.FullName(protoName))
		if err != nil {
			// ignore les non trouvés (logging utile)
			fmt.Println("FindDescriptorByName failed for", protoName, ":", err)
			continue
		}

		// desc est un protoreflect.MessageDescriptor (implémentation de Descriptor)
		md, ok := desc.(protoreflect.MessageDescriptor)
		if !ok {
			fmt.Println("descriptor cast failed for", protoName)
			continue
		}

		module := InferModuleFromTypeURL(typeURL)
		requiresAuth := mdHasAuthority(md)
		tmpl := buildJSONTemplateFromDescriptor(md)

		// Enrichir le template avec les valeurs actuelles si applicable
		if requiresAuth {
			currentValues := b.getCurrentValuesForMessage(b.ctx, module, protoName, tmpl)
			if currentValues != nil {
				fmt.Printf("Enriching template for %s with current values: %+v\n", protoName, currentValues)
				tmpl = enrichTemplateWithCurrentValues(tmpl, currentValues)
			} else {
				// Log pour déboguer si currentValues est nil
				if strings.Contains(protoName, "UpdateParams") {
					fmt.Printf("No current values found for %s (module: %s)\n", protoName, module)
				}
			}
		}

		entry := rpctypes.MsgCatalogEntry{
			TypeURL:       typeURL,
			ProtoFullName: protoName,
			Module:        module,
			Service:       "", // on peut remplir via protoregistry si nécessaire
			Method:        "",
			RequiresAuth:  requiresAuth,
			JSONTemplate:  tmpl,
		}
		out = append(out, entry)
	}

	return out, nil
}

// mdHasAuthority regarde si le message a un champ "authority" (insensible à la casse)
func mdHasAuthority(md protoreflect.MessageDescriptor) bool {
	flds := md.Fields()
	for i := 0; i < flds.Len(); i++ {
		f := flds.Get(i)
		if strings.EqualFold(string(f.Name()), "authority") {
			return true
		}
	}
	return false
}

// buildJSONTemplateFromDescriptor génère un map[string]interface{} récursif minimal
func buildJSONTemplateFromDescriptor(md protoreflect.MessageDescriptor) map[string]interface{} {
	out := make(map[string]interface{})
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		jsonName := string(f.JSONName())

		var val interface{}
		switch f.Kind() {
		case protoreflect.StringKind:
			val = ""
		case protoreflect.BoolKind:
			val = false
		case protoreflect.Int32Kind, protoreflect.Int64Kind,
			protoreflect.Uint32Kind, protoreflect.Uint64Kind:
			// Cosmos préfère parfois string pour big ints: on met "0"
			val = "0"
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			val = 0.0
		case protoreflect.BytesKind:
			val = "" // base64 attendu parfois
		case protoreflect.EnumKind:
			// on met le nom vide (le front peut lister les valeurs si besoin)
			val = ""
		case protoreflect.MessageKind:
			full := string(f.Message().FullName())
			// cas spéciaux Cosmos courants
			switch full {
			case "cosmos.base.v1beta1.Coin":
				val = map[string]string{"denom": "", "amount": "0"}
			case "cosmos.base.v1beta1.DecCoin":
				val = map[string]string{"denom": "", "amount": "0.0"}
			case "google.protobuf.Duration":
				val = "0s"
			case "google.protobuf.Timestamp":
				val = time.Now().UTC().Format(time.RFC3339)
			default:
				// récursif
				val = buildJSONTemplateFromDescriptor(f.Message())
			}
		default:
			val = nil
		}

		// gestion des repeated
		if f.Cardinality() == protoreflect.Repeated {
			// Pour les arrays de Coins, mettre un exemple plus clair
			if _, ok := val.(map[string]string); ok {
				// C'est un Coin, créer un exemple avec une valeur réaliste
				exampleCoin := map[string]string{
					"denom":  "ahelios",
					"amount": "1000000000000000000", // 1 HLS en wei
				}
				out[jsonName] = []interface{}{exampleCoin}
			} else {
				out[jsonName] = []interface{}{val}
			}
		} else {
			out[jsonName] = val
		}
	}
	return out
}

// func (b *Backend) GovCatalog(id uint64) (interface{}, error) {
// 	// impls := b.clientCtx.InterfaceRegistry.ListImplementations("cosmos.base.v1beta1.Msg")
// 	// for _, impl := range impls {
// 	// 	module := InferModuleFromTypeURL(impl)
// 	// 	if module == "" {
// 	// 		continue
// 	// 	}
// 	// 	requiresAuthority, err := b.MessageRequiresAuthority(impl)
// 	// 	if err != nil {
// 	// 		fmt.Println(impl, err.Error())
// 	// 		continue
// 	// 	}
// 	// 	fmt.Println(impl, module, requiresAuthority)
// 	// }

// 	b.DebugProtoRegistry([]string{
// 		"helios.hyperion.v1.MsgUpdateChainTokenLogo",
// 		"helios.hyperion.v1.MsgAddOneWhitelistedAddress",
// 		"cosmos.slashing.v1beta1.MsgUpdateParams",
// 	})
// 	return "", nil
// }

// getCurrentValuesForMessage récupère les valeurs actuelles pour un message donné
// Fonction générique qui détecte automatiquement le type de message
func (b *Backend) getCurrentValuesForMessage(ctx context.Context, module, protoName string, template map[string]interface{}) map[string]interface{} {
	// Détecter le type de message depuis le nom
	msgName := extractMessageName(protoName)
	
	// Pour MsgUpdateParams, récupérer les params actuels du module
	if strings.Contains(msgName, "UpdateParams") {
		params := b.getCurrentModuleParams(ctx, module)
		if params == nil {
			return nil
		}
		
		// Le template a une structure {"params": {...}}, donc wrapper les params
		// Vérifier si le template a une clé "params"
		if _, hasParams := template["params"]; hasParams {
			return map[string]interface{}{
				"params": params,
			}
		}
		
		// Sinon, retourner directement les params (pour les cas où la structure est différente)
		return params
	}
	
	// Pour d'autres types de messages, on pourrait ajouter d'autres stratégies ici
	// Par exemple : MsgSoftwareUpgrade -> récupérer le plan actuel
	// MsgCommunityPoolSpend -> récupérer le solde du community pool
	// etc.
	
	return nil
}

// getCurrentModuleParams récupère les paramètres actuels d'un module via gRPC
// Fonction générique qui utilise les clients QueryClient existants
func (b *Backend) getCurrentModuleParams(ctx context.Context, module string) map[string]interface{} {
	// Utiliser les clients QueryClient existants de manière générique
	// Convertir la réponse en JSON puis parser
	var paramsResp map[string]interface{}
	
	switch module {
	case "bank":
		res, err := b.queryClient.Bank.Params(ctx, &banktypes.QueryParamsRequest{})
		if err == nil {
			paramsBytes, _ := b.clientCtx.Codec.MarshalJSON(&res.Params)
			json.Unmarshal(paramsBytes, &paramsResp)
			paramsResp = convertKeysToCamelCase(paramsResp)
		}
	case "staking":
		res, err := b.queryClient.Staking.Params(ctx, &stakingtypes.QueryParamsRequest{})
		if err == nil {
			paramsBytes, _ := b.clientCtx.Codec.MarshalJSON(&res.Params)
			json.Unmarshal(paramsBytes, &paramsResp)
			paramsResp = convertKeysToCamelCase(paramsResp)
		}
	case "distribution":
		res, err := b.queryClient.Distribution.Params(ctx, &distributiontypes.QueryParamsRequest{})
		if err == nil {
			paramsBytes, _ := b.clientCtx.Codec.MarshalJSON(&res.Params)
			json.Unmarshal(paramsBytes, &paramsResp)
			paramsResp = convertKeysToCamelCase(paramsResp)
		}
	case "gov":
		res, err := b.queryClient.Gov.Params(ctx, &govtypes.QueryParamsRequest{ParamsType: "params"})
		if err == nil {
			// res.Params est déjà un pointeur pour gov
			paramsBytes, _ := b.clientCtx.Codec.MarshalJSON(res.Params)
			json.Unmarshal(paramsBytes, &paramsResp)
			paramsResp = convertKeysToCamelCase(paramsResp)
		}
	case "mint":
		res, err := b.queryClient.Mint.Params(ctx, &minttypes.QueryParamsRequest{})
		if err == nil {
			paramsBytes, _ := b.clientCtx.Codec.MarshalJSON(&res.Params)
			json.Unmarshal(paramsBytes, &paramsResp)
			paramsResp = convertKeysToCamelCase(paramsResp)
		}
	case "slashing":
		// Utiliser clientCtx pour créer un client slashing
		slashingClient := slashingtypes.NewQueryClient(b.clientCtx)
		res, err := slashingClient.Params(ctx, &slashingtypes.QueryParamsRequest{})
		if err != nil {
			// Log l'erreur pour déboguer
			fmt.Printf("Error fetching slashing params: %v\n", err)
			return nil
		}
		paramsBytes, marshalErr := b.clientCtx.Codec.MarshalJSON(&res.Params)
		if marshalErr != nil {
			fmt.Printf("Error marshaling slashing params: %v\n", marshalErr)
			return nil
		}
		if unmarshalErr := json.Unmarshal(paramsBytes, &paramsResp); unmarshalErr != nil {
			fmt.Printf("Error unmarshaling slashing params: %v\n", unmarshalErr)
			return nil
		}
		// Convertir les clés snake_case en camelCase pour correspondre au template
		paramsResp = convertKeysToCamelCase(paramsResp)
		// Log pour déboguer
		fmt.Printf("Slashing params retrieved (converted): %+v\n", paramsResp)
	case "logos":
		// Utiliser clientCtx pour créer un client logos
		logosClient := logostypes.NewQueryClient(b.clientCtx)
		res, err := logosClient.Params(ctx, &logostypes.QueryParamsRequest{})
		if err != nil {
			fmt.Printf("Error fetching logos params: %v\n", err)
			return nil
		}
		paramsBytes, marshalErr := b.clientCtx.Codec.MarshalJSON(&res.Params)
		if marshalErr != nil {
			fmt.Printf("Error marshaling logos params: %v\n", marshalErr)
			return nil
		}
		if unmarshalErr := json.Unmarshal(paramsBytes, &paramsResp); unmarshalErr != nil {
			fmt.Printf("Error unmarshaling logos params: %v\n", unmarshalErr)
			return nil
		}
		// Convertir les clés snake_case en camelCase pour correspondre au template
		paramsResp = convertKeysToCamelCase(paramsResp)
		fmt.Printf("Logos params retrieved (converted): %+v\n", paramsResp)
	case "consensus", "core":
		// Utiliser clientCtx pour créer un client consensus
		consensusClient := consensustypes.NewQueryClient(b.clientCtx)
		res, err := consensusClient.Params(ctx, &consensustypes.QueryParamsRequest{})
		if err != nil {
			fmt.Printf("Error fetching consensus params: %v\n", err)
			return nil
		}
		// Consensus params a une structure différente (ConsensusParams avec block, evidence, validator, abci)
		// On doit extraire les champs individuels
		if res.Params != nil {
			paramsResp = make(map[string]interface{})
			if res.Params.Block != nil {
				blockBytes, _ := b.clientCtx.Codec.MarshalJSON(res.Params.Block)
				var blockMap map[string]interface{}
				json.Unmarshal(blockBytes, &blockMap)
				paramsResp["block"] = convertKeysToCamelCase(blockMap)
			}
			if res.Params.Evidence != nil {
				evidenceBytes, _ := b.clientCtx.Codec.MarshalJSON(res.Params.Evidence)
				var evidenceMap map[string]interface{}
				json.Unmarshal(evidenceBytes, &evidenceMap)
				paramsResp["evidence"] = convertKeysToCamelCase(evidenceMap)
			}
			if res.Params.Validator != nil {
				validatorBytes, _ := b.clientCtx.Codec.MarshalJSON(res.Params.Validator)
				var validatorMap map[string]interface{}
				json.Unmarshal(validatorBytes, &validatorMap)
				paramsResp["validator"] = convertKeysToCamelCase(validatorMap)
			}
			if res.Params.Abci != nil {
				abciBytes, _ := b.clientCtx.Codec.MarshalJSON(res.Params.Abci)
				var abciMap map[string]interface{}
				json.Unmarshal(abciBytes, &abciMap)
				paramsResp["abci"] = convertKeysToCamelCase(abciMap)
			}
		}
		fmt.Printf("Consensus params retrieved (converted): %+v\n", paramsResp)
	// Ajouter d'autres modules au besoin
	default:
		// Pour les modules non supportés, retourner nil
		return nil
	}
	
	return paramsResp
}

// enrichTemplateWithCurrentValues enrichit le template avec les valeurs actuelles
// Structure : chaque champ peut avoir "_current" (valeur actuelle) et "_template" (valeur par défaut)
func enrichTemplateWithCurrentValues(template, currentValues map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Créer un index des clés currentValues avec normalisation (insensible à la casse et aux underscores)
	currentIndex := make(map[string]interface{})
	for k, v := range currentValues {
		normalized := normalizeKey(k)
		currentIndex[normalized] = v
		// Garder aussi la clé originale
		currentIndex[k] = v
	}
	
	for key, templateValue := range template {
		if key == "authority" || key == "@type" {
			result[key] = templateValue
			continue
		}
		
		// Chercher la valeur actuelle avec normalisation
		var currentValue interface{}
		var exists bool
		
		// Essayer d'abord avec la clé exacte
		if currentValue, exists = currentIndex[key]; !exists {
			// Essayer avec la clé normalisée
			normalizedKey := normalizeKey(key)
			currentValue, exists = currentIndex[normalizedKey]
		}
		
		if exists {
			// Si c'est un objet, merger récursivement
			if templateObj, ok := templateValue.(map[string]interface{}); ok {
				if currentObj, ok := currentValue.(map[string]interface{}); ok {
					result[key] = enrichTemplateWithCurrentValues(templateObj, currentObj)
				} else {
					// Objet dans template mais pas dans current, garder template
					result[key] = templateValue
				}
			} else {
				// Champ simple : ajouter template et current
				result[key] = map[string]interface{}{
					"_template": templateValue,
					"_current":  currentValue,
				}
			}
		} else {
			// Pas de valeur actuelle, garder le template
			result[key] = templateValue
		}
	}
	
	return result
}

// normalizeKey normalise une clé pour le matching (insensible à la casse et aux underscores)
func normalizeKey(key string) string {
	// Convertir en minuscule et remplacer les underscores
	normalized := strings.ToLower(key)
	normalized = strings.ReplaceAll(normalized, "_", "")
	return normalized
}

// convertKeysToCamelCase convertit les clés snake_case en camelCase récursivement
func convertKeysToCamelCase(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		camelKey := snakeToCamel(k)
		// Si la valeur est un map, convertir récursivement
		if nestedMap, ok := v.(map[string]interface{}); ok {
			result[camelKey] = convertKeysToCamelCase(nestedMap)
		} else if nestedArray, ok := v.([]interface{}); ok {
			// Si c'est un array, convertir chaque élément si c'est un map
			convertedArray := make([]interface{}, len(nestedArray))
			for i, item := range nestedArray {
				if itemMap, ok := item.(map[string]interface{}); ok {
					convertedArray[i] = convertKeysToCamelCase(itemMap)
				} else {
					convertedArray[i] = item
				}
			}
			result[camelKey] = convertedArray
		} else {
			result[camelKey] = v
		}
	}
	return result
}

// snakeToCamel convertit snake_case en camelCase
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		return s
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return result
}

// extractMessageName extrait le nom du message depuis le proto full name
func extractMessageName(protoFullName string) string {
	parts := strings.Split(protoFullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return protoFullName
}

// func (b *Backend) DebugProtoRegistry(targets []string) {

// 	for _, target := range targets {
// 		desc, err := b.clientCtx.Codec.InterfaceRegistry().FindDescriptorByName(protoreflect.FullName(target))
// 		if err != nil {
// 			fmt.Println("Error finding descriptor for", target, ":", err.Error())
// 			return
// 		}
// 		fmt.Println("Found descriptor for", target, ":", desc.FullName())
// 	}
// }
