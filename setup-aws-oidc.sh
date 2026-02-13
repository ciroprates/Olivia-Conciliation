#!/bin/bash
set -e

# Configuração
AWS_ACCOUNT_ID="683684736241"
AWS_REGION="us-east-1"
GITHUB_REPO="ciroprates/Olivia-Conciliation"
ROLE_NAME="GitHubActionsOliviaConciliationRole"
OIDC_PROVIDER_URL="token.actions.githubusercontent.com"

echo "🚀 Configurando GitHub Actions OIDC com AWS IAM Role"
echo "=================================================="
echo ""

# Passo 1: Criar OIDC Identity Provider (se não existir)
echo "📋 Passo 1: Verificando/Criando OIDC Identity Provider..."

PROVIDER_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${OIDC_PROVIDER_URL}"

if aws iam get-open-id-connect-provider --open-id-connect-provider-arn "$PROVIDER_ARN" 2>/dev/null; then
    echo "✅ OIDC Provider já existe: $PROVIDER_ARN"
else
    echo "🔧 Criando OIDC Provider..."
    
    # Obter thumbprint do GitHub (necessário para criar o provider)
    THUMBPRINT="6938fd4d98bab03faadb97b34396831e3780aea1"
    
    aws iam create-open-id-connect-provider \
        --url "https://${OIDC_PROVIDER_URL}" \
        --client-id-list "sts.amazonaws.com" \
        --thumbprint-list "$THUMBPRINT"
    
    echo "✅ OIDC Provider criado com sucesso!"
fi

echo ""

# Passo 2: Criar Trust Policy
echo "📋 Passo 2: Criando Trust Policy..."

cat > /tmp/trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "${PROVIDER_ARN}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${OIDC_PROVIDER_URL}:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "${OIDC_PROVIDER_URL}:sub": "repo:${GITHUB_REPO}:*"
        }
      }
    }
  ]
}
EOF

echo "✅ Trust Policy criada em /tmp/trust-policy.json"
echo ""

# Passo 3: Criar IAM Role
echo "📋 Passo 3: Criando IAM Role..."

if aws iam get-role --role-name "$ROLE_NAME" 2>/dev/null; then
    echo "⚠️  Role já existe. Atualizando trust policy..."
    aws iam update-assume-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-document file:///tmp/trust-policy.json
    echo "✅ Trust Policy atualizada!"
else
    echo "🔧 Criando nova role..."
    aws iam create-role \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document file:///tmp/trust-policy.json \
        --description "Role for GitHub Actions to deploy Olivia Conciliation app"
    echo "✅ Role criada com sucesso!"
fi

echo ""

# Passo 4: Criar e anexar política de permissões ECR
echo "📋 Passo 4: Configurando permissões ECR..."

cat > /tmp/ecr-permissions.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload"
      ],
      "Resource": "*"
    }
  ]
}
EOF

POLICY_NAME="GitHubActionsECRAccess"

# Verificar se a política já existe
if aws iam get-role-policy --role-name "$ROLE_NAME" --policy-name "$POLICY_NAME" 2>/dev/null; then
    echo "⚠️  Política inline já existe. Atualizando..."
else
    echo "🔧 Criando política inline..."
fi

aws iam put-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-name "$POLICY_NAME" \
    --policy-document file:///tmp/ecr-permissions.json

echo "✅ Permissões ECR configuradas!"
echo ""

# Passo 5: Exibir informações da Role
echo "📋 Passo 5: Informações da Role criada"
echo "=================================================="

ROLE_ARN=$(aws iam get-role --role-name "$ROLE_NAME" --query 'Role.Arn' --output text)

echo ""
echo "✅ Configuração concluída com sucesso!"
echo ""
echo "📝 Role ARN (use no GitHub Actions):"
echo "   $ROLE_ARN"
echo ""
echo "🔐 Trust Policy permite apenas:"
echo "   Repositório: $GITHUB_REPO"
echo "   Audience: sts.amazonaws.com"
echo ""
echo "🎯 Próximo passo:"
echo "   Atualize o workflow .github/workflows/ecr-push.yml"
echo "   para usar: role-to-assume: $ROLE_ARN"
echo ""

# Limpar arquivos temporários
rm -f /tmp/trust-policy.json /tmp/ecr-permissions.json

echo "✨ Script concluído!"
